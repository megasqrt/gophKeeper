package sqlite

import (
	"database/sql"
	"fmt"
)

// IsLoggedIn проверяет, сохранен ли токен.
func (s *SqliteStorage) IsLoggedIn() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, token, _, _, err := s.GetUserCredentials()
	return err == nil && token != ""
}

func (s *SqliteStorage) IsFirstRun() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.log.Info().Msg("Checking if first run")
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM config WHERE login IS NOT NULL").Scan(&count)
	if err != nil || count == 0 {
		s.log.Info().Msg("No user found in storage")
		return true
	}
	s.log.Info().Msg("User found in storage")
	return false
}

// LocalRegister "регистрирует" пользователя локально, сохраняя его имя и зашифрованный пароль.
func (s *SqliteStorage) LocalRegister(user, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, err := encriptPassword(user, password)
	if err != nil {
		return fmt.Errorf("could not derive key: %w", err)
	}
	s.key = key
	s.user = user

	encryptedPassword, err := s.encrypt([]byte(password))
	if err != nil {
		return fmt.Errorf("could not encrypt password for verification: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("INSERT OR REPLACE INTO config (login, user, password_hash) VALUES (?, ?, ?)", user, user, encryptedPassword); err != nil {
		return err
	}

	return tx.Commit()
}

// SaveUserCredentials сохраняет учетные данные пользователя.
func (s *SqliteStorage) SaveUserCredentials(login, token, deviceID string, encryptedMasterKey []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Шифруем чувствительные данные
	encryptedToken, err := s.encryptString(token)
	if err != nil {
		return fmt.Errorf("could not encrypt token: %w", err)
	}
	encryptedDeviceID, err := s.encryptString(deviceID)
	if err != nil {
		return fmt.Errorf("could not encrypt device ID: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("update config set login=?, token=?, device=?, encrypted_master_key=? where user=?",
		login, encryptedToken, encryptedDeviceID, encryptedMasterKey, s.user); err != nil {
		return err
	}

	return tx.Commit()
}

// GetUserCredentials извлекает учетные данные пользователя.
func (s *SqliteStorage) GetUserCredentials() (string, string, string, []byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.key == nil {
		return "", "", "", nil, fmt.Errorf("storage is locked, call Unlock first")
	}

	var login string
	var encryptedToken, encryptedDeviceID, encryptedMasterKey []byte

	err := s.db.QueryRow("SELECT login, token, device, encrypted_master_key FROM config WHERE login IS NOT NULL LIMIT 1").
		Scan(&login, &encryptedToken, &encryptedDeviceID, &encryptedMasterKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", nil, ErrUserNotRegistered
		}
		return "", "", "", nil, err
	}

	if len(encryptedToken) == 0 {
		return "", "", "", nil, fmt.Errorf("token is not set in database")
	}
	if len(encryptedDeviceID) == 0 {
		return "", "", "", nil, fmt.Errorf("device ID is not set in database")
	}

	token, err := s.decryptString(encryptedToken)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("could not decrypt token: %w", err)
	}

	deviceID, err := s.decryptString(encryptedDeviceID)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("could not decrypt device ID: %w", err)
	}

	return login, token, deviceID, encryptedMasterKey, nil
}