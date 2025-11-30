package storage

import (
	"go.etcd.io/bbolt"
)

// SaveText сохраняет текстовые данные в хранилище.
func (s *BboltStorage) SaveText(textData map[string]string) error {
	s.log.Info().Str("title", textData["title"]).Msg("Saving new text data")
	return s.saveItem(textBucket, StringMapToInterfaceMap(textData), true)
}

// UpdateText обновляет данные существующей текстовой записи.
func (s *BboltStorage) UpdateText(textData map[string]string) error {
	s.log.Info().Str("text_id", textData["id"]).Msg("Updating text data")
	return s.saveItem(textBucket, StringMapToInterfaceMap(textData), false)
}

// GetTexts извлекает все сохраненные текстовые записи.
func (s *BboltStorage) GetTexts() ([]map[string]string, error) {
	s.log.Info().Msg("Retrieving all texts from storage")
	items, err := s.getAllItems(textBucket)
	if err != nil {
		return nil, err
	}

	var texts []map[string]string
	for _, item := range items {
		texts = append(texts, InterfaceMapToStringMap(item))
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
