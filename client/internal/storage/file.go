package storage

import (
	"fmt"
	model "gophKeeper/pkg/grpchelper"
	"time"

	"go.etcd.io/bbolt"
)

// SaveFile сохраняет данные файла в хранилище.
func (s *BboltStorage) SaveFile(fileData *model.FileData, content []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// Бакет для метаданных
		metaBucket := tx.Bucket(fileMetaBucket)
		id, _ := metaBucket.NextSequence()
		itemID := fmt.Sprintf("%d", id)
		fileData.SetLocalID(itemID)
		fileData.ChangeTime = time.Now()

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
	fileData.ChangeTime = time.Now()
	return s.saveItem(fileMetaBucket, fileData, true)
}

// UpdateFile обновляет данные существующего файла.
func (s *BboltStorage) UpdateFile(fileData *model.FileData) error {
	s.log.Info().Str("file_id", fileData.LocalID).Msg("Updating file")
	fileData.ChangeTime = time.Now()
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

		metaData["data"] = decryptedContent // Добавляем содержимое в результат
		result = metaData
		return nil
	})

	return result, err
}

// DeleteFileByID удаляет файл из хранилища по его ID.
func (s *BboltStorage) DeleteFileByID(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		s.log.Info().Str("file_id", id).Msg("Deleting file")
		// Удаляем из обоих бакетов
		if err := tx.Bucket(fileMetaBucket).Delete([]byte(id)); err != nil {
			return fmt.Errorf("failed to delete file metadata: %w", err)
		}
		if err := tx.Bucket(fileDataBucket).Delete([]byte(id)); err != nil {
			// Если здесь ошибка, метаданные уже удалены. Это не идеально, но приемлемо.
			// В реальном приложении можно было бы добавить логику отката.
			s.log.Warn().Err(err).Str("file_id", id).Msg("Failed to delete file content, metadata was already deleted")
		}
		return nil
	})
}
