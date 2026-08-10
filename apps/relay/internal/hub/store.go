package hub

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/riffpad/riffpad/packages/protocol"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if err := db.AutoMigrate(&User{}, &OAuthAccount{}, &AuthToken{}, &HostRecord{}, &Device{}, &PairingRecord{}, &SessionMeta{}, &EmailOptout{}, &WaitlistEntry{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// AddWaitlistEmail records a waitlist signup. Duplicate subscriptions are a
// no-op so re-submitting the form does not create duplicate rows.
func (s *Store) AddWaitlistEmail(email string) error {
	e := &WaitlistEntry{Email: email, CreatedAt: time.Now()}
	return s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(e).Error
}

// WaitlistEmails returns all waitlist entries, oldest first.
func (s *Store) WaitlistEmails() ([]WaitlistEntry, error) {
	var rows []WaitlistEntry
	if err := s.db.Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// AddEmailOptout records an opt-out. Repeated unsubscribes are a no-op.
func (s *Store) AddEmailOptout(email string) error {
	o := &EmailOptout{Email: email, CreatedAt: time.Now()}
	return s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(o).Error
}

// EmailOptedOut reports whether an address has unsubscribed.
func (s *Store) EmailOptedOut(email string) (bool, error) {
	var n int64
	if err := s.db.Model(&EmailOptout{}).Where("email = ?", email).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// EmailOptouts returns all opted-out addresses, oldest first.
func (s *Store) EmailOptouts() ([]string, error) {
	var rows []EmailOptout
	if err := s.db.Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Email)
	}
	return out, nil
}

func (s *Store) CreateUser(username, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &User{ID: newID(), Username: username, PasswordHash: string(hash), CreatedAt: time.Now()}
	if err := s.db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) VerifyLogin(username, password string) (*User, error) {
	var u User
	if err := s.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, errors.New("invalid credentials")
	}
	return &u, nil
}

// FindOrCreateGitHubUser returns the user linked to a GitHub uid, creating
// the account (and a passwordless user) on first sign-in.
func (s *Store) FindOrCreateGitHubUser(ghID, ghLogin, email string) (*User, error) {
	var link OAuthAccount
	if err := s.db.Where("provider = ? AND provider_uid = ?", "github", ghID).First(&link).Error; err == nil {
		var u User
		if err := s.db.First(&u, "id = ?", link.UserID).Error; err != nil {
			return nil, err
		}
		return &u, nil
	}
	// Create a passwordless user with a unique username derived from the
	// GitHub login.
	username := ghLogin
	if username == "" {
		username = "gh-" + ghID[:8]
	}
	base := username
	for i := 0; i < 5; i++ {
		if err := s.db.Where("username = ?", username).First(&User{}).Error; err != nil {
			break
		}
		username = fmt.Sprintf("%s-%04d", base, time.Now().UnixNano()%10000)
	}
	u := &User{
		ID:       protocol.NewID(),
		Username: username,
		Email:    email,
	}
	if err := s.db.Create(u).Error; err != nil {
		return nil, err
	}
	link = OAuthAccount{
		ID:          protocol.NewID(),
		UserID:      u.ID,
		Provider:    "github",
		ProviderUID: ghID,
		CreatedAt:   time.Now(),
	}
	if err := s.db.Create(&link).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) CreateToken(userID string, ttl time.Duration) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	t := &AuthToken{
		ID: newID(), UserID: userID,
		TokenHash: hashToken(token), ExpiresAt: time.Now().Add(ttl), CreatedAt: time.Now(),
	}
	if err := s.db.Create(t).Error; err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) UserByToken(token string) (*User, error) {
	var t AuthToken
	if err := s.db.Where("token_hash = ?", hashToken(token)).First(&t).Error; err != nil {
		return nil, errors.New("invalid token")
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, errors.New("token expired")
	}
	var u User
	if err := s.db.First(&u, "id = ?", t.UserID).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return &u, nil
}

func (s *Store) DeleteToken(token string) error {
	return s.db.Where("token_hash = ?", hashToken(token)).Delete(&AuthToken{}).Error
}

func (s *Store) CreateHost(ownerID, name, secret string) (*HostRecord, error) {
	h := &HostRecord{ID: "h-" + newID()[:12], OwnerID: ownerID, Name: name, Secret: secret, CreatedAt: time.Now()}
	if err := s.db.Create(h).Error; err != nil {
		return nil, err
	}
	return h, nil
}

func (s *Store) GetHost(id string) (*HostRecord, error) {
	var h HostRecord
	if err := s.db.First(&h, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

// Pairing lookup/consumption outcomes.
var (
	ErrPairingInvalid = errors.New("invalid or expired pairing code")
	ErrPairingUsed    = errors.New("pairing code already used")
)

// CreatePairing stores a new pairing code, opportunistically purging expired
// rows so the table does not grow without bound.
func (s *Store) CreatePairing(p *PairingRecord) error {
	_ = s.db.Where("expires_at < ?", time.Now()).Delete(&PairingRecord{}).Error
	return s.db.Create(p).Error
}

// GetPairing returns the pairing row for code. Unknown and expired codes both
// yield ErrPairingInvalid; expired rows are deleted lazily.
func (s *Store) GetPairing(code string) (*PairingRecord, error) {
	var p PairingRecord
	if err := s.db.First(&p, "code = ?", code).Error; err != nil {
		return nil, ErrPairingInvalid
	}
	if time.Now().After(p.ExpiresAt) {
		_ = s.db.Delete(&p).Error
		return nil, ErrPairingInvalid
	}
	return &p, nil
}

// ConsumePairing atomically marks the code used. The conditional UPDATE is a
// compare-and-swap: concurrent consumers race on consumed_at IS NULL and only
// one wins; the losers get ErrPairingUsed.
func (s *Store) ConsumePairing(code string) error {
	res := s.db.Model(&PairingRecord{}).
		Where("code = ? AND consumed_at IS NULL", code).
		Update("consumed_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPairingUsed
	}
	return nil
}

func (s *Store) CreateDevice(ownerID, hostID, name, curve, pub string) (*Device, error) {
	d := &Device{ID: newID(), OwnerID: ownerID, HostID: hostID, Name: name, Curve: curve, PublicKey: pub, CreatedAt: time.Now()}
	if err := s.db.Create(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) GetDevice(id string) (*Device, error) {
	var d Device
	if err := s.db.First(&d, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// DevicesForUser lists all devices owned by the user.
func (s *Store) DevicesForUser(userID string) ([]Device, error) {
	var list []Device
	if err := s.db.Where("owner_id = ?", userID).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteDevice removes a device owned by the given user.
func (s *Store) DeleteDevice(id, ownerID string) error {
	return s.db.Where("id = ? AND owner_id = ?", id, ownerID).Delete(&Device{}).Error
}

// DeleteDevicesForHost removes every device paired to a host.
func (s *Store) DeleteDevicesForHost(hostID string) error {
	return s.db.Where("host_id = ?", hostID).Delete(&Device{}).Error
}

func (s *Store) HostIDsForUser(userID string) ([]string, error) {
	var ids []string
	if err := s.db.Model(&HostRecord{}).Where("owner_id = ?", userID).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// HostsForUser lists the hosts owned by a user (secrets excluded).
func (s *Store) HostsForUser(userID string) ([]HostRecord, error) {
	var list []HostRecord
	if err := s.db.Where("owner_id = ?", userID).Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *Store) SessionsForHosts(hostIDs []string) ([]SessionMeta, error) {
	var out []SessionMeta
	if len(hostIDs) == 0 {
		return out, nil
	}
	if err := s.db.Where("host_id IN ?", hostIDs).Order("last_seen_at desc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) UpsertSessions(hostID string, sessions []SessionMeta) error {
	now := time.Now()
	for i := range sessions {
		sessions[i].HostID = hostID
		if sessions[i].LastSeenAt.IsZero() {
			// Older daemons announce without a timestamp: fall back to "live
			// right now" instead of persisting Go's zero time.
			sessions[i].LastSeenAt = now
		}
		if err := s.db.Save(&sessions[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) MarkHostSessionsOffline(hostID string) error {
	return s.db.Model(&SessionMeta{}).Where("host_id = ?", hostID).Update("status", "offline").Error
}

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
