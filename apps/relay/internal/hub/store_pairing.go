package hub

import (
	"time"
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
