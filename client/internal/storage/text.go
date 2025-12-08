package storage

import (
	"crypto/sha256"
	"encoding/hex"
	model "gophKeeper/pkg/grpchelper"
	"time"
)

// SaveText сохраняет текстовые данные в хранилище.
func (s *BboltStorage) SaveText(textData *model.TextData) error {
	s.log.Info().Str("title", textData.Title).Msg("Saving new text data")
	textData.ChangeTime = time.Now()
	h := sha256.New()
	h.Write([]byte(textData.Text))
	textData.Checksum = hex.EncodeToString(h.Sum(nil))
	return s.saveItem(textBucket, textData, true)
}

// UpdateText обновляет данные существующей текстовой записи.
func (s *BboltStorage) UpdateText(textData *model.TextData) error {
	s.log.Info().Str("text_id", textData.LocalID).Msg("Updating text data")
	textData.ChangeTime = time.Now()
	h := sha256.New()
	h.Write([]byte(textData.Text))
	textData.Checksum = hex.EncodeToString(h.Sum(nil))
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

// DeleteText удаляет текстовую запись по ID.
func (s *BboltStorage) DeleteText(id string) error {
	return s.markAsDeleted(textBucket, id, func() interface{} {
		return &model.TextData{}
	})
}
