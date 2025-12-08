package storage

import (
	"gophKeeper/client/internal/domain/model"
	"time"

	"go.etcd.io/bbolt"
)

// SaveText сохраняет текстовые данные в хранилище.
func (s *BboltStorage) SaveText(textData *model.TextData) error {
	s.log.Info().Str("title", textData.Title).Msg("Saving new text data")
	textData.ChangeTime = time.Now()
	return s.saveItem(textBucket, textData, true)
}

// UpdateText обновляет данные существующей текстовой записи.
func (s *BboltStorage) UpdateText(textData *model.TextData) error {
	s.log.Info().Str("text_id", textData.LocalID).Msg("Updating text data")
	textData.ChangeTime = time.Now()
	return s.saveItem(textBucket, textData, false)
}

// GetTexts извлекает все сохраненные текстовые записи.
func (s *BboltStorage) GetTexts() ([]model.TextData, error) {
	s.log.Info().Msg("Retrieving all texts from storage")
	items, err := s.getAllItems(textBucket, func(data map[string]interface{}) (interface{}, error) {
		return model.FromMapText(data)
	})
	if err != nil {
		return nil, err
	}

	var texts []model.TextData
	for _, item := range items {
		texts = append(texts, item.(model.TextData))
	}
	return texts, nil
}

// DeleteText удаляет текстовую запись по ID.
func (s *BboltStorage) DeleteText(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(textBucket)
		s.log.Info().Str("text_id", id).Msg("Deleting text data")
		return b.Delete([]byte(id))
	})
}
