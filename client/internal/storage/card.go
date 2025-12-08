package storage

import (
	"crypto/sha256"
	"encoding/hex"
	model "gophKeeper/pkg/grpchelper"
	"time"

	"go.etcd.io/bbolt"
)

// calculateCardChecksum вычисляет checksum для карты на основе всех полей данных.
func calculateCardChecksum(card *model.Card) string {
	h := sha256.New()
	h.Write([]byte(card.Number))
	h.Write([]byte(card.Holder))
	h.Write([]byte(card.Expiry))
	h.Write([]byte(card.CVV))
	return hex.EncodeToString(h.Sum(nil))
}

// SaveCard сохраняет данные карты в хранилище.
func (s *BboltStorage) SaveCard(cardData *model.Card) error {
	s.log.Info().Msg("Saving new card")
	cardData.ChangeTime = time.Now()
	cardData.Checksum = calculateCardChecksum(cardData)
	return s.saveItem(cardsBucket, cardData, true)
}

// UpdateCard обновляет данные существующей карты.
func (s *BboltStorage) UpdateCard(cardData *model.Card) error {
	s.log.Info().Str("card_id", cardData.LocalID).Msg("Updating card")
	cardData.ChangeTime = time.Now()
	cardData.Checksum = calculateCardChecksum(cardData)
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

// GetShortCards извлекает краткую информацию о картах для синхронизации.
func (s *BboltStorage) GetShortCards() ([]model.SyncInfo, error) {
	s.log.Info().Msg("Retrieving all info cards from storage")
	var syncInfos []model.SyncInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cardsBucket)
		return b.ForEach(func(k, v []byte) error {
			var card model.Card
			if err := s.decryptItem(v, &card); err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not decrypt card for sync info")
				return nil // Пропускаем поврежденные записи
			}

			syncInfos = append(syncInfos, model.SyncInfo{
				LocalID:  card.LocalID,
				ServerID: card.ServerID,
				Checksum: card.Checksum,
				Deleted:  card.Deleted,
			})
			return nil
		})
	})

	return syncInfos, err
}

// GetCardsByIDs извлекает карты по их идентификаторам.
func (s *BboltStorage) GetCardsByIDs(ids []string) ([]model.Card, error) {
	s.log.Info().Int("count", len(ids)).Msg("Retrieving cards by IDs from storage")
	var cards []model.Card

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cardsBucket)
		for _, id := range ids {
			encryptedData := b.Get([]byte(id))
			if encryptedData == nil {
				s.log.Warn().Str("card_id", id).Msg("Card not found, skipping")
				continue
			}

			var card model.Card
			if err := s.decryptItem(encryptedData, &card); err != nil {
				s.log.Error().Str("card_id", id).Err(err).Msg("Failed to decrypt card")
				continue
			}

			cards = append(cards, card)
		}
		return nil
	})

	return cards, err
}

// DeleteCard помечает карту как удаленную.
func (s *BboltStorage) DeleteCard(id string) error {
	return s.markAsDeleted(cardsBucket, id, func() interface{} {
		return &model.Card{}
	})
}
