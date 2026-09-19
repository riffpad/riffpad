package hub

import (
	"errors"
	"gorm.io/gorm"
	"time"
)

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

// GetSessionClientMeta returns the client-side meta for one session, or nil
// when none exists yet.
func (s *Store) GetSessionClientMeta(sessionID, hostID string) (*SessionClientMeta, error) {
	var m SessionClientMeta
	err := s.db.Where("session_id = ? AND host_id = ?", sessionID, hostID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpsertSessionClientMeta saves the full client-side meta for a session.
func (s *Store) UpsertSessionClientMeta(m SessionClientMeta) error {
	m.UpdatedAt = time.Now()
	return s.db.Save(&m).Error
}

// SessionClientMetaForUser lists all client-side session meta for an account.
func (s *Store) SessionClientMetaForUser(userID string) ([]SessionClientMeta, error) {
	var out []SessionClientMeta
	if err := s.db.Where("user_id = ?", userID).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
