package hub

import (
	"time"
)

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
