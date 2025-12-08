package storage

import (
	model "gophKeeper/pkg/grpchelper"
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
	items, err := s.getAllItems(cardsBucket, func() interface{} {
		return &model.Card{}
	})
	if err != nil {
		return nil, err
	}

	var cards []model.Card
	for _, item := range items {
		if card, ok := item.(*model.Card); ok {
			cards = append(cards, *card)
		}
	}
	return cards, nil
}

func (s *BboltStorage) GetShortCards() ([]model.SyncInfo, error) {
	s.log.Info().Msg("Retrieving all info cards from storage")
	items, err := s.GetShortSyncInfo(cardsBucket)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// GetCardsByIDs извлекает карты по их идентификаторам.
func (s *BboltStorage) GetCardsByIDs(ids []string) ([]model.Card, error) {
	s.log.Info().Int("count", len(ids)).Msg("Retrieving cards by IDs from storage")
	var cards []model.Card

	for _, id := range ids {
		itemData, err := s.getItemByID(cardsBucket, id)
		if err != nil {
			s.log.Warn().Str("card_id", id).Err(err).Msg("Card not found, skipping")
			continue
		}

		card, err := model.FromMapCard(itemData)
		if err != nil {
			s.log.Error().Str("card_id", id).Err(err).Msg("Failed to convert map to card model")
			continue
		}

		cards = append(cards, card)
	}

	return cards, nil
}

// DeleteCard помечает карту как удаленную.
func (s *BboltStorage) DeleteCard(id string) error {
	return s.markAsDeleted(cardsBucket, id, func() interface{} {
		return &model.Card{}
	})
}
