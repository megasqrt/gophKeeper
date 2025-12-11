package storage

import (
	"crypto/sha256"
	"encoding/hex"
	model "gophKeeper/pkg/grpchelper"
	"time"

	"go.etcd.io/bbolt"
)

// calculateTextChecksum вычисляет checksum для текстовой записи на основе всех полей данных.
func calculateTextChecksum(text *model.TextData) string {
	h := sha256.New()
	h.Write([]byte(text.Title))
	h.Write([]byte(text.Text))
	return hex.EncodeToString(h.Sum(nil))
}

// SaveText сохраняет текстовые данные в хранилище.
func (s *BboltStorage) SaveText(textData *model.TextData) error {
	s.log.Info().Str("title", textData.Title).Msg("Saving new text data")
	textData.ChangeTime = time.Now().Unix()
	textData.Checksum = calculateTextChecksum(textData)
	return s.saveItem(textBucket, textData, true)
}

// UpdateText обновляет данные существующей текстовой записи.
func (s *BboltStorage) UpdateText(textData *model.TextData) error {
	s.log.Info().Str("text_id", textData.LocalID).Msg("Updating text data")
	textData.ChangeTime = time.Now().Unix()
	textData.Checksum = calculateTextChecksum(textData)
	return s.saveItem(textBucket, textData, false)
}

// GetTexts извлекает все сохраненные текстовые записи.
func (s *BboltStorage) GetTexts() ([]model.TextData, error) {
	s.log.Info().Msg("Retrieving all texts from storage")
	items, err := s.getAllItems(textBucket, func() interface{} {
		return &model.TextData{}
	})
	if err != nil {
		return nil, err
	}

	var texts []model.TextData
	for _, item := range items {
		if text, ok := item.(*model.TextData); ok {
			texts = append(texts, *text)
		}
	}
	return texts, nil
}

// GetShortTexts извлекает краткую информацию о текстах для синхронизации.
func (s *BboltStorage) GetShortTexts() ([]model.SyncInfo, error) {
	s.log.Info().Msg("Retrieving all info texts from storage")
	var syncInfos []model.SyncInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(textBucket)
		return b.ForEach(func(k, v []byte) error {
			var text model.TextData
			if err := s.decryptItem(v, &text); err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not decrypt text for sync info")
				return nil // Пропускаем поврежденные записи
			}

			syncInfos = append(syncInfos, model.SyncInfo{
				LocalID:  text.LocalID,
				ServerID: text.ServerID,
				Checksum: text.Checksum,
				Deleted:  text.Deleted,
			})
			return nil
		})
	})

	return syncInfos, err
}

// GetTextsByIDs извлекает тексты по их идентификаторам.
func (s *BboltStorage) GetTextsByIDs(ids []string) ([]model.TextData, error) {
	s.log.Info().Int("count", len(ids)).Msg("Retrieving texts by IDs from storage")
	var texts []model.TextData

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(textBucket)
		for _, id := range ids {
			encryptedData := b.Get([]byte(id))
			if encryptedData == nil {
				s.log.Warn().Str("text_id", id).Msg("Text not found, skipping")
				continue
			}

			var text model.TextData
			if err := s.decryptItem(encryptedData, &text); err != nil {
				s.log.Error().Str("text_id", id).Err(err).Msg("Failed to decrypt text")
				continue
			}

			texts = append(texts, text)
		}
		return nil
	})

	return texts, err
}

// DeleteText помечает текстовую запись как удаленную.
func (s *BboltStorage) DeleteText(id string) error {
	return s.markAsDeleted(textBucket, id, func() interface{} {
		return &model.TextData{}
	})
}

// DeleteHardText физически удаляет карту из хранилища.
func (s *BboltStorage) DeleteHardText(id string) error {
	return s.deleteItem(textBucket, id)
}