package storage

import "time"

// SaveCard сохраняет данные карты в хранилище.
func (s *BboltStorage) SaveCard(cardData map[string]string) error {
	s.log.Info().Msg("Saving new card")
	cardData["changeTime"] = time.Now().Format(time.RFC3339Nano)
	return s.saveItem(cardsBucket, StringMapToInterfaceMap(cardData), true)
}

// UpdateCard обновляет данные существующей карты.
func (s *BboltStorage) UpdateCard(cardData map[string]string) error {
	s.log.Info().Str("card_id", cardData["id"]).Msg("Updating card")
	cardData["changeTime"] = time.Now().Format(time.RFC3339Nano)
	return s.saveItem(cardsBucket, StringMapToInterfaceMap(cardData), false)
}

// GetCards извлекает все сохраненные карты.
func (s *BboltStorage) GetCards() ([]map[string]string, error) {
	s.log.Info().Msg("Retrieving all cards from storage")
	items, err := s.getAllItems(cardsBucket)
	if err != nil {
		return nil, err
	}

	// Конвертируем обратно в []map[string]string
	var cards []map[string]string
	for _, item := range items {
		cards = append(cards, InterfaceMapToStringMap(item))
	}
	return cards, nil
}
