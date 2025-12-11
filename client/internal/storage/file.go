package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	model "gophKeeper/pkg/grpchelper"
	"time"

	"go.etcd.io/bbolt"
)

// calculateFileChecksum вычисляет checksum для файла на основе метаданных (Name, Size, Metadata).
// Содержимое файла не включается в checksum для экономии ресурсов.
func calculateFileChecksum(file *model.FileData) string {
	h := sha256.New()
	h.Write([]byte(file.Name))
	h.Write([]byte(fmt.Sprintf("%d", file.Size)))
	h.Write([]byte(file.Metadata))
	return hex.EncodeToString(h.Sum(nil))
}

// SaveFile сохраняет данные файла в хранилище.
func (s *BboltStorage) SaveFile(fileData *model.FileData, content []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// Бакет для метаданных
		metaBucket := tx.Bucket(fileMetaBucket)
		id, _ := metaBucket.NextSequence()
		itemID := fmt.Sprintf("%d", id)
		fileData.SetLocalID(itemID)
		fileData.ChangeTime = time.Now().Unix()
		fileData.Checksum = calculateFileChecksum(fileData)

		// 1. Сохраняем метаданные (без содержимого)
		err := s.saveItemTx(tx, fileMetaBucket, fileData, true)
		if err != nil {
			return fmt.Errorf("could not save file metadata: %w", err)
		}

		// 2. Сохраняем содержимое файла в отдельный бакет
		dataBucket := tx.Bucket(fileDataBucket)
		encryptedData, err := s.encrypt(content)
		if err != nil {
			return fmt.Errorf("could not encrypt file data: %w", err)
		}
		return dataBucket.Put([]byte(itemID), encryptedData)
	})
}

// SaveFileMetadata сохраняет только метаданные файла. Используется при синхронизации.
func (s *BboltStorage) SaveFileMetadata(fileData *model.FileData) error {
	s.log.Info().Str("file_name", fileData.Name).Msg("Saving file metadata")
	fileData.ChangeTime = time.Now().Unix()
	fileData.Checksum = calculateFileChecksum(fileData)
	return s.saveItem(fileMetaBucket, fileData, true)
}

// UpdateFile обновляет данные существующего файла.
func (s *BboltStorage) UpdateFile(fileData *model.FileData) error {
	s.log.Info().Str("file_id", fileData.LocalID).Msg("Updating file")
	fileData.ChangeTime = time.Now().Unix()
	fileData.Checksum = calculateFileChecksum(fileData)
	return s.saveItem(fileMetaBucket, fileData, false)
}

// GetFiles извлекает метаданные всех сохраненных файлов.
// Обратите внимание: для экономии памяти этот метод не загружает содержимое файлов (поле "data").
func (s *BboltStorage) GetFiles() ([]model.FileData, error) {
	s.log.Info().Msg("Retrieving all files from storage")
	// Читаем только метаданные
	items, err := s.getAllItems(fileMetaBucket, func() interface{} {
		return &model.FileData{}
	})
	if err != nil {
		return nil, err
	}
	var files []model.FileData
	for _, item := range items {
		if file, ok := item.(*model.FileData); ok {
			files = append(files, *file)
		}
	}
	return files, nil
}

// GetFileByID извлекает один файл по ID, включая его содержимое.
func (s *BboltStorage) GetFileByID(id string) (map[string]interface{}, error) {
	s.log.Info().Str("file_id", id).Msg("Retrieving file from storage")
	var result map[string]interface{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		// 1. Получаем метаданные
		metaBucket := tx.Bucket(fileMetaBucket)
		metaData, err := s.getItemByIDTx(metaBucket, id)
		if err != nil {
			return err
		}

		// 2. Получаем содержимое файла
		dataBucket := tx.Bucket(fileDataBucket)
		encryptedContent := dataBucket.Get([]byte(id))
		if encryptedContent == nil {
			// Это может произойти, если файл был создан через синхронизацию без содержимого
			s.log.Warn().Str("file_id", id).Msg("File content not found locally")
			result = metaData
			return nil
		}

		decryptedContent, err := s.decrypt(encryptedContent)
		if err != nil {
			return fmt.Errorf("could not decrypt file content for id '%s': %w", id, err)
		}

		// Создаем новый map, чтобы избежать проблем с типами при добавлении []byte
		// metaData уже десериализован из JSON, поэтому добавляем данные напрямую
		result = make(map[string]interface{})
		for k, v := range metaData {
			result[k] = v
		}
		result["data"] = decryptedContent // Добавляем содержимое как []byte
		return nil
	})

	return result, err
}

// GetShortFiles извлекает краткую информацию о файлах для синхронизации.
func (s *BboltStorage) GetShortFiles() ([]model.SyncInfo, error) {
	s.log.Info().Msg("Retrieving all info files from storage")
	var syncInfos []model.SyncInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(fileMetaBucket)
		return b.ForEach(func(k, v []byte) error {
			var file model.FileData
			if err := s.decryptItem(v, &file); err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not decrypt file for sync info")
				return nil // Пропускаем поврежденные записи
			}

			syncInfos = append(syncInfos, model.SyncInfo{
				LocalID:  file.LocalID,
				ServerID: file.ServerID,
				Checksum: file.Checksum,
				Deleted:  file.Deleted,
			})
			return nil
		})
	})

	return syncInfos, err
}

// GetFilesByIDs извлекает метаданные файлов по их идентификаторам.
func (s *BboltStorage) GetFilesByIDs(ids []string) ([]model.FileData, error) {
	s.log.Info().Int("count", len(ids)).Msg("Retrieving files by IDs from storage")
	var files []model.FileData

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(fileMetaBucket)
		for _, id := range ids {
			encryptedData := b.Get([]byte(id))
			if encryptedData == nil {
				s.log.Warn().Str("file_id", id).Msg("File not found, skipping")
				continue
			}

			var file model.FileData
			if err := s.decryptItem(encryptedData, &file); err != nil {
				s.log.Error().Str("file_id", id).Err(err).Msg("Failed to decrypt file")
				continue
			}

			files = append(files, file)
		}
		return nil
	})

	return files, err
}

// DeleteFileByID помечает файл как удаленный (soft delete).
// Для физического удаления используется отдельный метод по команде пользователя из TUI.
func (s *BboltStorage) DeleteFileByID(id string) error {
	return s.markAsDeleted(fileMetaBucket, id, func() interface{} {
		return &model.FileData{}
	})
}
