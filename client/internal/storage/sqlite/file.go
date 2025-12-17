package sqlite

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	model "gophKeeper/pkg/grpchelper"
	"database/sql"
	"time"

)

// calculateFileChecksum вычисляет checksum для файла
func calculateFileChecksum(name string, size int64, metadata string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte(fmt.Sprintf("%d", size)))
	h.Write([]byte(metadata))
	return hex.EncodeToString(h.Sum(nil))
}


func (s *SqliteStorage) GetFiles() ([]model.FileData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, server_id, name, metadata, size, checksum, created_at, updated_at, deleted_at, version 
		FROM binary_data 
		WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.FileData
	for rows.Next() {
		var f model.FileData
		var encryptedName, encryptedMetadata []byte
		var deletedAt sql.NullInt64
		var createdAt int64

		if err := rows.Scan(&f.LocalID, &encryptedName, &encryptedMetadata, &f.Size,
			&f.Checksum, &createdAt, &f.ChangeTime, &deletedAt); err != nil {
			s.log.Error().Err(err).Msg("Failed to scan file")
			continue
		}

		f.Name, err = s.decryptString(encryptedName)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to decrypt file name")
			continue
		}
		if len(encryptedMetadata) > 0 {
			f.Metadata, err = s.decryptString(encryptedMetadata)
			if err != nil {
				s.log.Error().Err(err).Msg("Failed to decrypt metadata")
				continue
			}
		}
		f.Deleted = deletedAt.Valid

		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *SqliteStorage) GetFilesByIDs(ids []int64) ([]model.FileData, error) {
	return nil, nil
}

func (s *SqliteStorage) GetFileByID(id int64) (map[string]interface{}, error) {
	return nil, nil
}

func (s *SqliteStorage) UpdateFile(data *model.FileData) error {
	return s.SaveFileMetadata(data)
}

func (s *SqliteStorage) SaveFile(data *model.FileData, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	data.ChangeTime = now
	data.Checksum = calculateFileChecksum(data.Name, data.Size, data.Metadata)

	encryptedName, err := s.encryptString(data.Name)
	if err != nil {
		return fmt.Errorf("could not encrypt file name: %w", err)
	}

	var encryptedMetadata []byte
	if data.Metadata != "" {
		encryptedMetadata, err = s.encryptString(data.Metadata)
		if err != nil {
			return fmt.Errorf("could not encrypt metadata: %w", err)
		}
	}

	encryptedContent, err := s.encrypt(content)
	if err != nil {
		return fmt.Errorf("could not encrypt file content: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO binary_data 
		(server_id, data, name, metadata, size, checksum, created_at, updated_at, deleted_at, version) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		data.ServerID, encryptedContent, encryptedName, encryptedMetadata,
		data.Size, data.Checksum, now, now,
		sql.NullInt64{Valid: data.Deleted, Int64: now}, data.Version)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SqliteStorage) SaveFileMetadata(data *model.FileData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	data.ChangeTime = now
	data.Checksum = calculateFileChecksum(data.Name, data.Size, data.Metadata)

	encryptedName, err := s.encryptString(data.Name)
	if err != nil {
		return fmt.Errorf("could not encrypt file name: %w", err)
	}

	var encryptedMetadata []byte
	if data.Metadata != "" {
		encryptedMetadata, err = s.encryptString(data.Metadata)
		if err != nil {
			return fmt.Errorf("could not encrypt metadata: %w", err)
		}
	}

	_, err = s.db.Exec(`
		UPDATE binary_data 
		SET name = ?, metadata = ?, size = ?, checksum = ?, updated_at = ?, server_id = ?,
		version = ?
		WHERE id = ?`,
		encryptedName, encryptedMetadata, data.Size, data.Checksum, now, data.ServerID, data.Version,  data.LocalID)
	return err
}

func (s *SqliteStorage) DeleteFileByID(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE binary_data SET deleted_at = ? WHERE id = ?", now, id)
	return err
}