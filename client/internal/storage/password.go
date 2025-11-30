package storage

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

// SavePass сохраняет данные пароля в хранилище.
func (s *BboltStorage) SavePass(passData map[string]string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(passwordsBucket)

		// Генерируем уникальный ID для пароля.
		id, _ := b.NextSequence()
		passData["id"] = fmt.Sprintf("%d", id)

		// Сериализуем данные пароля в JSON
		jsonData, err := json.Marshal(passData)
		if err != nil {
			return fmt.Errorf("could not marshal password data: %w", err)
		}

		// Шифруем JSON
		encryptedData, err := s.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("could not encrypt password data: %w", err)
		}

		s.log.Info().Str("pass_id", passData["id"]).Msg("Saving new password")
		return b.Put([]byte(passData["id"]), encryptedData)
	})
}

// UpdatePass обновляет данные существующего пароля.
func (s *BboltStorage) UpdatePass(passData map[string]string) error {
	passID, ok := passData["id"]
	if !ok || passID == "" {
		return errors.New("password ID is missing for update")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(passwordsBucket)

		jsonData, err := json.Marshal(passData)
		if err != nil {
			return fmt.Errorf("could not marshal password data for update: %w", err)
		}

		encryptedData, err := s.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("could not encrypt password data for update: %w", err)
		}

		s.log.Info().Str("pass_id", passID).Msg("Updating password")
		return b.Put([]byte(passID), encryptedData)
	})
}

// GetPasss извлекает все сохраненные пароли.
func (s *BboltStorage) GetPasss() ([]map[string]string, error) {
	var passwords []map[string]string
	s.log.Info().Msg("Retrieving all passwords from storage")
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(passwordsBucket)
		return b.ForEach(func(k, v []byte) error {
			decryptedData, err := s.decrypt(v)
			if err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not decrypt password data")
				return fmt.Errorf("could not decrypt password data for key %s: %w", k, err)
			}
			var passData map[string]string
			if err := json.Unmarshal(decryptedData, &passData); err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not unmarshal password data")
				return fmt.Errorf("could not unmarshal password data for key %s: %w", k, err)
			}
			passwords = append(passwords, passData)
			return nil
		})
	})
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get passwords from storage")
	}
	return passwords, err
}
