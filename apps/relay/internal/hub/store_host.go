package hub

import (
	"time"
)

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

func (s *Store) MarkHostSessionsOffline(hostID string) error {
	return s.db.Model(&SessionMeta{}).Where("host_id = ?", hostID).Update("status", "offline").Error
}
