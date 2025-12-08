package storage

import (
	"encoding/json"
	"fmt"
	"gophKeeper/client/internal/domain/model"
	"time"

	"go.etcd.io/bbolt"
)

// SaveFile сохраняет данные файла в хранилище.
func (s *BboltStorage) SaveFile(fileData *model.FileData, content []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(binaryBucket)
		id, _ := b.NextSequence()
		itemID := fmt.Sprintf("%d", id)
		fileData.SetLocalID(itemID)
		fileData.ChangeTime = time.Now()

		// Преобразуем метаданные в map
		itemMap := fileData.ToMap()
		// Добавляем бинарные данные
		itemMap["data"] = content

		jsonData, err := json.Marshal(itemMap)
		if err != nil {
			return fmt.Errorf("could not marshal file data: %w", err)
		}

		encryptedData, err := s.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("could not encrypt file data: %w", err)
		}
		return b.Put([]byte(itemID), encryptedData)
	})
}

// SaveFileMetadata сохраняет только метаданные файла. Используется при синхронизации.
func (s *BboltStorage) SaveFileMetadata(fileData *model.FileData) error {
	s.log.Info().Str("file_name", fileData.Name).Msg("Saving file metadata")
	fileData.ChangeTime = time.Now()
	return s.saveItem(binaryBucket, fileData, true)
}

// UpdateFile обновляет данные существующего файла.
func (s *BboltStorage) UpdateFile(fileData *model.FileData) error {
	s.log.Info().Str("file_id", fileData.LocalID).Msg("Updating file")
	fileData.ChangeTime = time.Now()
	return s.saveItem(binaryBucket, fileData, false)
}

// GetFiles извлекает метаданные всех сохраненных файлов.
// Обратите внимание: для экономии памяти этот метод не загружает содержимое файлов (поле "data").
func (s *BboltStorage) GetFiles() ([]model.FileData, error) {
	s.log.Info().Msg("Retrieving all files from storage")
	items, err := s.getAllItems(binaryBucket, func(data map[string]interface{}) (interface{}, error) {
		return model.FromMapFile(data)
	})
	if err != nil {
		return nil, err
	}
	var files []model.FileData
	for _, item := range items {
		files = append(files, item.(model.FileData))
	}
	return files, nil
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
