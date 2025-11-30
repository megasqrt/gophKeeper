package storage

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

// SaveFile сохраняет данные файла в хранилище.
// Ключом является имя файла. Если файл с таким именем уже существует, он будет перезаписан.
func (s *BboltStorage) SaveFile(data map[string]interface{}) error {
	name, ok := data["name"].(string)
	if !ok || name == "" {
		return errors.New("file name is missing in data map")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(binaryBucket)

		// Генерируем уникальный ID для файла.
		id, _ := b.NextSequence()
		data["id"] = fmt.Sprintf("%d", id)

		// Сериализуем данные файла в JSON.
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("could not marshal file data for '%s': %w", name, err)
		}

		// Шифруем JSON.
		encryptedData, err := s.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("could not encrypt file data for '%s': %w", name, err)
		}

		s.log.Info().Str("file_name", name).Msg("Saving file")
		return b.Put([]byte(data["id"].(string)), encryptedData)
	})
}

// GetFiles извлекает метаданные всех сохраненных файлов.
// Обратите внимание: для экономии памяти этот метод не загружает содержимое файлов (поле "data").
func (s *BboltStorage) GetFiles() ([]map[string]interface{}, error) {
	var files []map[string]interface{}
	s.log.Info().Msg("Retrieving all files from storage")

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(binaryBucket)
		return b.ForEach(func(k, v []byte) error {
			decryptedData, err := s.decrypt(v)
			if err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not decrypt file data")
				return fmt.Errorf("could not decrypt file data for key %s: %w", k, err)
			}

			var fileData map[string]interface{}
			if err := json.Unmarshal(decryptedData, &fileData); err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not unmarshal file data")
				return fmt.Errorf("could not unmarshal file data for key %s: %w", k, err)
			}

			// Удаляем содержимое файла, чтобы не загружать его в память при листинге.
			delete(fileData, "data")

			files = append(files, fileData)
			return nil
		})
	})

	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get files from storage")
	}
	return files, err
}

// GetFileByID извлекает один файл по ID, включая его содержимое.
func (s *BboltStorage) GetFileByID(id string) (map[string]interface{}, error) {
	var fileData map[string]interface{}
	s.log.Info().Str("file_id", id).Msg("Retrieving file from storage")

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(binaryBucket)
		encryptedData := b.Get([]byte(id))
		if encryptedData == nil {
			return fmt.Errorf("file with id '%s' not found", id)
		}

		decryptedData, err := s.decrypt(encryptedData)
		if err != nil {
			return fmt.Errorf("could not decrypt file data for id '%s': %w", id, err)
		}

		if err := json.Unmarshal(decryptedData, &fileData); err != nil {
			return fmt.Errorf("could not unmarshal file data for id '%s': %w", id, err)
		}
		return nil
	})

	return fileData, err
}

// DeleteFileByID удаляет файл из хранилища по его ID.
func (s *BboltStorage) DeleteFileByID(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(binaryBucket)
		s.log.Info().Str("file_id", id).Msg("Deleting file")
		return b.Delete([]byte(id))
	})
}
