package storage

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

// SaveText сохраняет текстовые данные в хранилище.
func (s *BboltStorage) SaveText(textData map[string]string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(textBucket)

		// Генерируем уникальный ID для текста.
		id, _ := b.NextSequence()
		textData["id"] = fmt.Sprintf("%d", id)

		// Сериализуем данные в JSON
		jsonData, err := json.Marshal(textData)
		if err != nil {
			return fmt.Errorf("could not marshal text data: %w", err)
		}

		// Шифруем JSON
		encryptedData, err := s.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("could not encrypt text data: %w", err)
		}

		s.log.Info().Str("text_id", textData["id"]).Str("title", textData["title"]).Msg("Saving new text data")
		return b.Put([]byte(textData["id"]), encryptedData)
	})
}

// UpdateText обновляет данные существующей текстовой записи.
func (s *BboltStorage) UpdateText(textData map[string]string) error {
	textID, ok := textData["id"]
	if !ok || textID == "" {
		return errors.New("text ID is missing for update")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(textBucket)

		jsonData, err := json.Marshal(textData)
		if err != nil {
			return fmt.Errorf("could not marshal text data for update: %w", err)
		}

		encryptedData, err := s.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("could not encrypt text data for update: %w", err)
		}

		s.log.Info().Str("text_id", textID).Msg("Updating text data")
		return b.Put([]byte(textID), encryptedData)
	})
}

// GetTexts извлекает все сохраненные текстовые записи.
func (s *BboltStorage) GetTexts() ([]map[string]string, error) {
	var texts []map[string]string
	s.log.Info().Msg("Retrieving all texts from storage")
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(textBucket)
		return b.ForEach(func(k, v []byte) error {
			decryptedData, err := s.decrypt(v)
			if err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not decrypt text data")
				return fmt.Errorf("could not decrypt text data for key %s: %w", k, err)
			}
			var textData map[string]string
			if err := json.Unmarshal(decryptedData, &textData); err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not unmarshal text data")
				return fmt.Errorf("could not unmarshal text data for key %s: %w", k, err)
			}
			s.log.Debug().Str("json", string(decryptedData)).Msg("Decrypted text data")
			texts = append(texts, textData)
			return nil
		})
	})
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get texts from storage")
	}
	return texts, err
}

// DeleteText удаляет текстовую запись по ID.
func (s *BboltStorage) DeleteText(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(textBucket)
		s.log.Info().Str("text_id", id).Msg("Deleting text data")
		return b.Delete([]byte(id))
	})
}
