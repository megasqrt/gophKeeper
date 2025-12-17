package sqlite

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	models "gophKeeper/pkg/grpchelper"
	"time"
)

// calculateCardChecksum вычисляет checksum для карты на основе всех полей данных.
func calculateCardChecksum(number, holder, expiry, cvv, metadata string) string {
	h := sha256.New()
	h.Write([]byte(number))
	h.Write([]byte(holder))
	h.Write([]byte(expiry))
	h.Write([]byte(cvv))
	h.Write([]byte(metadata))
	return hex.EncodeToString(h.Sum(nil))
}

// cardDataStruct структура для маршалинга/анмаршалинга данных карты
type cardDataStruct struct {
	Number   string `json:"number"`
	Holder   string `json:"holder"`
	Expiry   string `json:"expiry"`
	CVV      string `json:"cvv"`
	Metadata string `json:"metadata"`
}

// encryptCardData шифрует данные карты
func (s *SqliteStorage) encryptCardData(cardData *models.Card) ([]byte, error) {
	data := cardDataStruct{
		Number:   cardData.Number,
		Holder:   cardData.Holder,
		Expiry:   cardData.Expiry,
		CVV:      cardData.CVV,
		Metadata: cardData.Metadata,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal card data: %w", err)
	}

	encryptedData, err := s.encrypt(jsonData)
	if err != nil {
		return nil, fmt.Errorf("could not encrypt card data: %w", err)
	}

	return encryptedData, nil
}

// decryptCardData расшифровывает данные карты
func (s *SqliteStorage) decryptCardData(encryptedData []byte) (*cardDataStruct, error) {
	decryptedData, err := s.decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt card data: %w", err)
	}

	var data cardDataStruct
	if err := json.Unmarshal(decryptedData, &data); err != nil {
		return nil, fmt.Errorf("could not unmarshal card data: %w", err)
	}

	return &data, nil
}

func (s *SqliteStorage) SaveCard(cardData *models.Card) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	if cardData.ChangeTime == 0 {
		cardData.ChangeTime = now
	}
	cardData.ChangeTime = now
	var createTime int64 = cardData.ChangeTime
	cardData.Checksum = calculateCardChecksum(cardData.Number, cardData.Holder, cardData.Expiry, cardData.CVV, cardData.Metadata)

	// Шифруем данные карты
	encryptedData, err := s.encryptCardData(cardData)
	if err != nil {
		return err
	}

	// Используем транзакцию для атомарности
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	var deletedAt sql.NullInt64
	if cardData.Deleted {
		deletedAt = sql.NullInt64{Valid: true, Int64: now}
	}

	// INSERT новой записи
	var serverID sql.NullString
	if cardData.ServerID != 0 {
		serverID = sql.NullString{Valid: true, String: fmt.Sprintf("%d", cardData.ServerID)}
	}

	_, err = tx.Exec(`
		INSERT INTO cards 
		(server_id, data, checksum, created_at, updated_at, deleted_at, version) 
		VALUES (?, ?, ?, ?, ?, ?, 0)`,
		serverID, encryptedData, cardData.Checksum, createTime, cardData.ChangeTime, deletedAt, cardData.Version)
	if err != nil {
		return fmt.Errorf("could not save card: %w", err)
	}

	return tx.Commit()
}

func (s *SqliteStorage) UpdateCard(cardData *models.Card) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем наличие LocalID
	if cardData.LocalID == 0 {
		return errors.New("LocalID is required for update")
	}

	now := time.Now().Unix()
	cardData.ChangeTime = now
	cardData.Checksum = calculateCardChecksum(cardData.Number, cardData.Holder, cardData.Expiry, cardData.CVV, cardData.Metadata)

	// Шифруем данные карты
	encryptedData, err := s.encryptCardData(cardData)
	if err != nil {
		return err
	}

	// Используем транзакцию для атомарности
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	var deletedAt sql.NullInt64
	if cardData.Deleted {
		deletedAt = sql.NullInt64{Valid: true, Int64: now}
	}

	// UPDATE существующей записи
	var serverID sql.NullString
	if cardData.ServerID != 0 {
		serverID = sql.NullString{Valid: true, String: fmt.Sprintf("%d", cardData.ServerID)}
	}

	_, err = tx.Exec(`
		UPDATE cards 
		SET data = ?, checksum = ?, updated_at = ?, deleted_at = ?, server_id = ?, version = ?
		WHERE id = ?`,
		encryptedData, cardData.Checksum, cardData.ChangeTime, deletedAt, serverID, cardData.Version, cardData.LocalID)
	if err != nil {
		return fmt.Errorf("could not update card: %w", err)
	}

	return tx.Commit()
}

func (s *SqliteStorage) GetCards() ([]models.Card, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, server_id, data, checksum, created_at, updated_at, deleted_at, version
		FROM cards 
		WHERE deleted_at IS NULL
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("could not query cards: %w", err)
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		var id int64
		var serverID sql.NullString
		var encryptedData []byte
		var checksum string
		var createTime, updateTime int64
		var deletedAt sql.NullInt64
        var version int32
		if err := rows.Scan(&id, &serverID, &encryptedData, &checksum, &createTime, &updateTime, &deletedAt, &version); err != nil {
			s.log.Error().Err(err).Msg("Failed to scan card")
			continue
		}

		// Расшифровываем данные карты
		data, err := s.decryptCardData(encryptedData)
		if err != nil {
			s.log.Error().Err(err).Int64("id", id).Msg("Failed to decrypt card data")
			continue
		}

		// Парсим ServerID
		var parsedServerID int64
		if serverID.Valid && serverID.String != "" {
			if _, err := fmt.Sscanf(serverID.String, "%d", &parsedServerID); err != nil {
				parsedServerID = 0
			}
		}

		// Собираем модель из метаданных (колонки) и данных (JSON)
		c := models.Card{
			LocalID:     id,
			ServerID:    parsedServerID,
			Number:      data.Number,
			Holder:      data.Holder,
			Expiry:      data.Expiry,
			CVV:         data.CVV,
			Metadata:    data.Metadata,
			Checksum:    checksum,
			ChangeTime:  updateTime,
			Deleted:     deletedAt.Valid,
			Version:     version,	
		}

		cards = append(cards, c)
	}

	return cards, rows.Err()
}

func (s *SqliteStorage) GetCardsByIDs(ids []int64) ([]models.Card, error) {
	return nil, nil
}

func (s *SqliteStorage) DeleteCard(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE cards SET deleted_at = ? WHERE id = ?", now, id)
	return err
}

// FindCardByServerID находит карту по ServerID
func (s *SqliteStorage) FindCardByServerID(serverID int64) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var localID int64
	serverIDStr := fmt.Sprintf("%d", serverID)
	err := s.db.QueryRow("SELECT id FROM cards WHERE server_id = ? AND deleted_at IS NULL LIMIT 1", serverIDStr).Scan(&localID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // Не найдено
		}
		return 0, err
	}
	return localID, nil
}