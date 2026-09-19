package hub

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// User is a relay account. Hosts, devices and sessions are owned by a user.
type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex" json:"username"`
	PasswordHash string    `json:"-"`
	Email        string    `json:"email,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// OAuthAccount links an external provider identity (e.g. GitHub) to a user.
type OAuthAccount struct {
	ID          string `gorm:"primaryKey"`
	UserID      string `gorm:"index"`
	Provider    string `gorm:"uniqueIndex:idx_provider_uid,priority:1"`
	ProviderUID string `gorm:"uniqueIndex:idx_provider_uid,priority:2"`
	CreatedAt   time.Time
}

// AuthToken is an opaque bearer token (stored as SHA-256).
type AuthToken struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"index"`
	TokenHash string `gorm:"uniqueIndex"`
	ExpiresAt time.Time
	CreatedAt time.Time
}

// HostRecord is a registered host owned by a user.
type HostRecord struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	OwnerID   string    `gorm:"index" json:"ownerId"`
	Name      string    `json:"name"`
	Secret    string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
}

// Device is a paired viewer device owned by a user and bound to a host.
type Device struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	OwnerID   string    `gorm:"index" json:"ownerId"`
	HostID    string    `gorm:"index" json:"hostId"`
	Name      string    `json:"name"`
	Curve     string    `json:"curve"`
	PublicKey string    `json:"publicKey"`
	CreatedAt time.Time `json:"createdAt"`
}

// PairingRecord is a one-time pairing code issued for a host. Codes live in
// the database so a relay restart does not invalidate in-flight pairings, and
// ConsumedAt enforces single use (a code can pair exactly one device).
type PairingRecord struct {
	Code       string `gorm:"primaryKey"`
	HostID     string `gorm:"index"`
	Curve      string
	PublicKey  string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// SessionMeta is the metadata of a session announced by a host.
type SessionMeta struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	HostID     string    `gorm:"index" json:"hostId"`
	Name       string    `json:"name"`
	CLI        string    `json:"cli"`
	Cwd        string    `json:"cwd"`
	Status     string    `json:"status"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

// SessionClientMeta is client-side session metadata (custom display name and
// hidden state) owned by the account. It is layered on top of the
// host-announced SessionMeta and never sent back to the host: renaming or
// hiding a session only changes what clients see, never what the agent does.
type SessionClientMeta struct {
	SessionID   string    `gorm:"primaryKey" json:"sessionId"`
	HostID      string    `gorm:"primaryKey" json:"hostId"`
	UserID      string    `gorm:"index" json:"userId"`
	DisplayName string    `json:"displayName"`
	Hidden      bool      `json:"hidden"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// EmailOptout is a waitlist address that asked to stop receiving
// announcement emails. The address is normalized (lower-cased) and stored as
// the primary key so unsubscribe is idempotent.
type EmailOptout struct {
	Email     string    `gorm:"primaryKey" json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// WaitlistEntry is an email collected by the landing-page form. Stored in
// the relay database so announcement tooling can pull the list directly
// instead of depending on a third-party form backend.
type WaitlistEntry struct {
	Email     string    `gorm:"primaryKey" json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type Store struct {
	db *gorm.DB
}

// OpenStore opens the metadata store. When databaseURL is non-empty it uses
// Postgres; otherwise it falls back to a SQLite file in dataDir.
func OpenStore(dataDir, databaseURL string) (*Store, error) {
	var (
		db  *gorm.DB
		err error
	)
	if databaseURL != "" {
		db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	} else {
		if err := os.MkdirAll(dataDir, 0o700); err != nil {
			return nil, err
		}
		db, err = gorm.Open(sqlite.Open(filepath.Join(dataDir, "relay.db")+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"), &gorm.Config{})
	}
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&User{}, &OAuthAccount{}, &AuthToken{}, &HostRecord{}, &Device{}, &PairingRecord{}, &SessionMeta{}, &SessionClientMeta{}, &EmailOptout{}, &WaitlistEntry{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Pairing lookup/consumption outcomes.
var (
	ErrPairingInvalid = errors.New("invalid or expired pairing code")
	ErrPairingUsed    = errors.New("pairing code already used")
)

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
