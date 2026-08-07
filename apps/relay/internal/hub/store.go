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
	if err := db.AutoMigrate(&User{}, &OAuthAccount{}, &AuthToken{}, &HostRecord{}, &Device{}, &SessionMeta{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
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
	for i := range sessions {
		sessions[i].HostID = hostID
		sessions[i].LastSeenAt = time.Now()
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
