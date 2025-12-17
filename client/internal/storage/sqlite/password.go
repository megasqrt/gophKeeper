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

// calculatePasswordChecksum вычисляет checksum для пароля на основе всех полей данных.
func calculatePasswordChecksum(login, password, description string) string {
	h := sha256.New()
	h.Write([]byte(login))
	h.Write([]byte(password))
	h.Write([]byte(description))
	return hex.EncodeToString(h.Sum(nil))
}

// passwordData структура для маршалинга/анмаршалинга данных пароля
type passwordData struct {
	Login       string `json:"login"`
	Password    string `json:"password"`
	Description string `json:"description"`
}

// encryptPasswordData шифрует данные пароля
func (s *SqliteStorage) encryptPasswordData(passData *models.Password) ([]byte, error) {
	data := passwordData{
		Login:       passData.Login,
		Password:    passData.Password,
		Description: passData.Description,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("could not marshal password data: %w", err)
	}

	encryptedData, err := s.encrypt(jsonData)
	if err != nil {
		return nil, fmt.Errorf("could not encrypt password data: %w", err)
	}

	return encryptedData, nil
}

// decryptPasswordData расшифровывает данные пароля
func (s *SqliteStorage) decryptPasswordData(encryptedData []byte) (*passwordData, error) {
	decryptedData, err := s.decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt password data: %w", err)
	}

	var data passwordData
	if err := json.Unmarshal(decryptedData, &data); err != nil {
		return nil, fmt.Errorf("could not unmarshal password data: %w", err)
	}

	return &data, nil
}

func (s *SqliteStorage) SavePass(passData *models.Password) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	if passData.CreateTime == 0 {
		passData.CreateTime = now
	}
	passData.ChangeTime = now
	passData.Checksum = calculatePasswordChecksum(passData.Login, passData.Password, passData.Description)

	// Шифруем данные пароля
	encryptedData, err := s.encryptPasswordData(passData)
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
	if passData.Deleted {
		deletedAt = sql.NullInt64{Valid: true, Int64: now}
	}

	// INSERT новой записи
	var serverID sql.NullString
	if passData.ServerID != 0 {
		serverID = sql.NullString{Valid: true, String: fmt.Sprintf("%d", passData.ServerID)}
	}

	_, err = tx.Exec(`
		INSERT INTO credentials 
		(server_id, data, checksum, created_at, updated_at, deleted_at, client_version) 
		VALUES (?, ?, ?, ?, ?, ?, 0)`,
		serverID, encryptedData, passData.Checksum, passData.CreateTime, passData.ChangeTime, deletedAt)
	if err != nil {
		return fmt.Errorf("could not save password: %w", err)
	}

	return tx.Commit()
}

func (s *SqliteStorage) UpdatePass(passData *models.Password) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем наличие LocalID
	if passData.LocalID == 0 {
		return errors.New("LocalID is required for update")
	}

	now := time.Now().Unix()
	passData.ChangeTime = now
	passData.Checksum = calculatePasswordChecksum(passData.Login, passData.Password, passData.Description)

	// Шифруем данные пароля
	encryptedData, err := s.encryptPasswordData(passData)
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
	if passData.Deleted {
		deletedAt = sql.NullInt64{Valid: true, Int64: now}
	}

	// UPDATE существующей записи
	_, err = tx.Exec(`
		UPDATE credentials 
		SET data = ?, checksum = ?, updated_at = ?, deleted_at = ?, server_id = ?
		WHERE id = ?`,
		encryptedData, passData.Checksum, passData.ChangeTime, deletedAt, passData.ServerID, passData.LocalID)
	if err != nil {
		return fmt.Errorf("could not update password: %w", err)
	}

	return tx.Commit()
}

func (s *SqliteStorage) GetPasswords() ([]models.Password, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, server_id, data, checksum, created_at, updated_at, deleted_at 
		FROM credentials 
		WHERE deleted_at IS NULL
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("could not query passwords: %w", err)
	}
	defer rows.Close()

	var passwords []models.Password
	for rows.Next() {
		var id int64
		var serverID sql.NullString
		var encryptedData []byte
		var checksum string
		var createTime, updateTime int64
		var deletedAt sql.NullInt64

		if err := rows.Scan(&id, &serverID, &encryptedData, &checksum, &createTime, &updateTime, &deletedAt); err != nil {
			s.log.Error().Err(err).Msg("Failed to scan password")
			continue
		}

		// Расшифровываем данные пароля
		data, err := s.decryptPasswordData(encryptedData)
		if err != nil {
			s.log.Error().Err(err).Int64("id", id).Msg("Failed to decrypt password data")
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
		p := models.Password{
			LocalID:     id,
			ServerID:    parsedServerID,
			Login:       data.Login,
			Password:    data.Password,
			Description: data.Description,
			Checksum:    checksum,
			CreateTime:  createTime,
			ChangeTime:  updateTime,
			Deleted:     deletedAt.Valid,
		}

		passwords = append(passwords, p)
	}

	return passwords, rows.Err()
}

func (s *SqliteStorage) DeletePass(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE credentials SET deleted_at = ? WHERE id = ?", now, id)
	return err
}

// FindPasswordByServerID находит пароль по ServerID
func (s *SqliteStorage) FindPasswordByServerID(serverID int64) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var localID int64
	serverIDStr := fmt.Sprintf("%d", serverID)
	err := s.db.QueryRow("SELECT id FROM credentials WHERE server_id = ? AND deleted_at IS NULL LIMIT 1", serverIDStr).Scan(&localID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // Не найдено
		}
		return 0, err
	}
	return localID, nil
}
