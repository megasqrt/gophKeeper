package storage

import (
	"go.etcd.io/bbolt"
)

// SaveFile сохраняет данные файла в хранилище.
func (s *BboltStorage) SaveFile(data map[string]interface{}) error {
	s.log.Info().Str("file_name", data["name"].(string)).Msg("Saving file")
	return s.saveItem(binaryBucket, data, true)
}

// GetFiles извлекает метаданные всех сохраненных файлов.
// Обратите внимание: для экономии памяти этот метод не загружает содержимое файлов (поле "data").
func (s *BboltStorage) GetFiles() ([]map[string]interface{}, error) {
	s.log.Info().Msg("Retrieving all files from storage")
	return s.getAllItems(binaryBucket)
}

// GetFileByID извлекает один файл по ID, включая его содержимое.
func (s *BboltStorage) GetFileByID(id string) (map[string]interface{}, error) {
	s.log.Info().Str("file_id", id).Msg("Retrieving file from storage")
	return s.getItemByID(binaryBucket, id)
}

// DeleteFileByID удаляет файл из хранилища по его ID.
func (s *BboltStorage) DeleteFileByID(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(binaryBucket)
		s.log.Info().Str("file_id", id).Msg("Deleting file")
		return b.Delete([]byte(id))
	})
}
