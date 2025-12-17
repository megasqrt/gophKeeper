package sqlite

import (
	"crypto/sha256"
	"encoding/hex"
	model "gophKeeper/pkg/grpchelper"
	"fmt"
	"time"
	"database/sql"

)

// calculateTextChecksum вычисляет checksum для текста
func calculateTextChecksum(title, text string) string {
	h := sha256.New()
	h.Write([]byte(title))
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *SqliteStorage) SaveText(textData *model.TextData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	textData.ChangeTime = now
	textData.Checksum = calculateTextChecksum(textData.Title, textData.Text)

	// Шифруем чувствительные данные
	encryptedTitle, err := s.encryptString(textData.Title)
	if err != nil {
		return fmt.Errorf("could not encrypt title: %w", err)
	}
	encryptedText, err := s.encryptString(textData.Text)
	if err != nil {
		return fmt.Errorf("could not encrypt text: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO note 
		(server_id, title, text, checksum, created_at, updated_at, deleted_at, version) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		textData.ServerID, encryptedTitle, encryptedText, textData.Checksum,
		now, now, sql.NullInt64{Valid: textData.Deleted, Int64: now},textData.Version)
	return err
}

func (s *SqliteStorage) UpdateText(textData *model.TextData) error {
	return s.SaveText(textData)
}

func (s *SqliteStorage) GetTexts() ([]model.TextData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, server_id, title, text, checksum, created_at, updated_at, deleted_at, version 
		FROM note 
		WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var texts []model.TextData
	for rows.Next() {
		var t model.TextData
		var encryptedTitle, encryptedText []byte
		var deletedAt sql.NullInt64
		var createdAt int64

		if err := rows.Scan(&t.LocalID, &encryptedTitle, &encryptedText, &t.Checksum,
			&createdAt, &t.ChangeTime, &deletedAt); err != nil {
			s.log.Error().Err(err).Msg("Failed to scan text")
			continue
		}

		t.Title, err = s.decryptString(encryptedTitle)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to decrypt title")
			continue
		}
		t.Text, err = s.decryptString(encryptedText)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to decrypt text")
			continue
		}
		t.Deleted = deletedAt.Valid

		texts = append(texts, t)
	}
	return texts, rows.Err()
}

func (s *SqliteStorage) GetTextsByIDs(ids []int64) ([]model.TextData, error) {
	return nil, nil
}

func (s *SqliteStorage) DeleteText(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE note SET deleted_at = ? WHERE id = ?", now, id)
	return err
}