package hub

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/riffpad/riffpad/packages/protocol"
	"golang.org/x/crypto/bcrypt"
	"time"
)

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
