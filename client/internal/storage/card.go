package storage

import (
	"encoding/json"
	"errors"
	"fmt"

	//"github.com/rs/zerolog"
	"go.etcd.io/bbolt"

)

// SaveCard сохраняет данные карты в хранилище.
func (s *BboltStorage) SaveCard(cardData map[string]string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cardsBucket)

		// Генерируем уникальный ID для карты.
		id, _ := b.NextSequence()
		cardData["id"] = fmt.Sprintf("%d", id)

		// Сериализуем данные карты в JSON
		jsonData, err := json.Marshal(cardData)
		if err != nil {
			return fmt.Errorf("could not marshal card data: %w", err)
		}

		// Шифруем JSON
		encryptedData, err := s.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("could not encrypt card data: %w", err)
		}

		s.log.Info().Str("card_id", cardData["id"]).Msg("Saving new card")
		return b.Put([]byte(cardData["id"]), encryptedData)
	})
}

// UpdateCard обновляет данные существующей карты.
func (s *BboltStorage) UpdateCard(cardData map[string]string) error {
	cardID, ok := cardData["id"]
	if !ok || cardID == "" {
		return errors.New("card ID is missing for update")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cardsBucket)

		jsonData, err := json.Marshal(cardData)
		if err != nil {
			return fmt.Errorf("could not marshal card data for update: %w", err)
		}

		encryptedData, err := s.encrypt(jsonData)
		if err != nil {
			return fmt.Errorf("could not encrypt card data for update: %w", err)
		}

		s.log.Info().Str("card_id", cardID).Msg("Updating card")
		return b.Put([]byte(cardID), encryptedData)
	})
}

// GetCards извлекает все сохраненные карты.
func (s *BboltStorage) GetCards() ([]map[string]string, error) {
	var cards []map[string]string
	s.log.Info().Msg("Retrieving all cards from storage")
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cardsBucket)
		return b.ForEach(func(k, v []byte) error {
			decryptedData, err := s.decrypt(v)
			if err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not decrypt card data")
				return fmt.Errorf("could not decrypt card data for key %s: %w", k, err)
			}
			var cardData map[string]string
			if err := json.Unmarshal(decryptedData, &cardData); err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not unmarshal card data")
				return fmt.Errorf("could not unmarshal card data for key %s: %w", k, err)
			}
			cards = append(cards, cardData)
			return nil
		})
	})
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get cards from storage")
	}
	return cards, err
}