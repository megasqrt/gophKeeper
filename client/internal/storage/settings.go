package storage

import(
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

// SaveLastSyncTime сохраняет время последней успешной синхронизации с сервером.
func (s *BboltStorage) SaveLastSyncTime(t time.Time) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		// Сериализуем time.Time в JSON, затем шифруем
		timeJSON, err := json.Marshal(t)
		if err != nil {
			return fmt.Errorf("could not marshal sync time: %w", err)
		}
		encryptedTime, err := s.encrypt(timeJSON)
		if err != nil {
			return fmt.Errorf("could not encrypt sync time: %w", err)
		}
		return b.Put(lastSyncKey, encryptedTime)
	})
}

// GetLastSyncTime извлекает время последней успешной синхронизации.
func (s *BboltStorage) GetLastSyncTime() (time.Time, error) {
	var t time.Time
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		timeBytes := b.Get(lastSyncKey)
		if timeBytes == nil {
			return nil // Время еще не сохранено
		}
		decryptedTime, err := s.decrypt(timeBytes)
		if err != nil {
			return fmt.Errorf("could not decrypt sync time: %w", err)
		}
		// Десериализуем из JSON
		if err := json.Unmarshal(decryptedTime, &t); err != nil {
			return fmt.Errorf("could not unmarshal sync time: %w", err)
		}
		return nil
	})
	return t, err
}