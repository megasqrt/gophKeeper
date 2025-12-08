package storage

import (
	"gophKeeper/client/internal/domain/model"
	"time"
)

// SaveCard сохраняет данные карты в хранилище.
func (s *BboltStorage) SaveCard(cardData *model.Card) error {
	s.log.Info().Msg("Saving new card")
	cardData.ChangeTime = time.Now()
	return s.saveItem(cardsBucket, cardData, true)
}

// UpdateCard обновляет данные существующей карты.
func (s *BboltStorage) UpdateCard(cardData *model.Card) error {
	s.log.Info().Str("card_id", cardData.LocalID).Msg("Updating card")
	cardData.ChangeTime = time.Now()
	return s.saveItem(cardsBucket, cardData, false)
}

// GetCards извлекает все сохраненные карты.
func (s *BboltStorage) GetCards() ([]model.Card, error) {
	s.log.Info().Msg("Retrieving all cards from storage")
	items, err := s.getAllItems(cardsBucket, func(data map[string]interface{}) (interface{}, error) {
		return model.FromMapCard(data)
	})
	if err != nil {
		return nil, err
	}

	var cards []model.Card
	for _, item := range items {
		cards = append(cards, item.(model.Card))
	}
	return cards, nil
}

// DeleteCard помечает карту как удаленную.
func (s *BboltStorage) DeleteCard(id string) error {
	return s.markAsDeleted(cardsBucket, id, func(data map[string]interface{}) (interface{}, error) {
		return model.FromMapCard(data)
	})
}
