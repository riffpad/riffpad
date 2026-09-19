package hub

import (
	"gorm.io/gorm/clause"
	"time"
)

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
