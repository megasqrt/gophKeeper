package storage

import (
	"fmt"

	"go.etcd.io/bbolt"
)

// LocalRegister сохраняет пароль для локального пользователя и инициализирует ключ шифрования.
func (s *BboltStorage) LocalRegister(user, password string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		s.log.Info().Msg("Deriving encryption key from password")
		key, err := encriptPassword(user, password)
		if err != nil {
			return fmt.Errorf("could not derive key: %w", err)
		}
		s.key = key
		encryptedPassword, err := s.encrypt([]byte(password))
		if err != nil {
			return fmt.Errorf("could not encrypt password: %w", err)
		}
		encryptedUser, err := s.encrypt([]byte(user))
		if err != nil {
			return fmt.Errorf("could not encrypt user: %w", err)
		}
		if err = b.Put(userKey, encryptedUser); err != nil {
			s.log.Error().Err(err).Msg("Failed to save login")
		}

		return b.Put(passwordKey, encryptedPassword)
	})
}

var masterKeyKey = []byte("master_key")

// SaveUserCredentials сохраняет токен, удаленный логин, ID устройства и зашифрованный мастер-ключ пользователя.
func (s *BboltStorage) SaveUserCredentials(login, token, deviceID string, encryptedMasterKey []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)

		// Сохраняем токен
		encryptedToken, err := s.encrypt([]byte(token))
		if err != nil {
			return fmt.Errorf("could not encrypt token: %w", err)
		}
		if err := b.Put(tokenKey, encryptedToken); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		// Сохраняем и удаленный логин, чтобы не запрашивать его каждый раз
		encryptedLogin, err := s.encrypt([]byte(login))
		if err != nil {
			return fmt.Errorf("could not encrypt remote login: %w", err)
		}
		if err := b.Put(loginKey, encryptedLogin); err != nil {
			return fmt.Errorf("failed to save remote login: %w", err)
		}

		// Сохраняем ID устройства
		encryptedDeviceID, err := s.encrypt([]byte(deviceID))
		if err != nil {
			return fmt.Errorf("could not encrypt device id: %w", err)
		}
		if err := b.Put(deviceKey, encryptedDeviceID); err != nil {
			return fmt.Errorf("failed to save device id: %w", err)
		}

		// Сохраняем зашифрованный мастер-ключ (зашифрованный локальным ключом)
		if len(encryptedMasterKey) > 0 {
			encryptedMasterKeyLocal, err := s.encrypt(encryptedMasterKey)
			if err != nil {
				return fmt.Errorf("could not encrypt master key: %w", err)
			}
			if err := b.Put(masterKeyKey, encryptedMasterKeyLocal); err != nil {
				return fmt.Errorf("failed to save master key: %w", err)
			}
		}

		return nil
	})
}

// GetUserCredentials извлекает удаленный логин, токен, ID устройства и зашифрованный мастер-ключ пользователя.
func (s *BboltStorage) GetUserCredentials() (login, token, device string, encryptedMasterKey []byte, err error) {
	err = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)

		tokenBytes := b.Get(tokenKey)
		if len(tokenBytes) > 0 {
			decryptedToken, err := s.decrypt(tokenBytes)
			if err != nil {
				// Если не можем расшифровать, возможно, пароль неверный
				return fmt.Errorf("could not decrypt token: %w", err)
			}
			token = string(decryptedToken)
		}
		loginBytes := b.Get(loginKey)
		if len(loginBytes) > 0 {
			decryptedLogin, err := s.decrypt(loginBytes)
			if err != nil {
				return fmt.Errorf("could not decrypt login: %w", err)
			}
			login = string(decryptedLogin)
		}

		deviceBytes := b.Get(deviceKey)
		if len(deviceBytes) > 0 {
			decryptedDevice, err := s.decrypt(deviceBytes)
			if err != nil {
				return fmt.Errorf("could not decrypt device: %w", err)
			}
			device = string(decryptedDevice)
		}

		masterKeyBytes := b.Get(masterKeyKey)
		if len(masterKeyBytes) > 0 {
			decryptedMasterKey, err := s.decrypt(masterKeyBytes)
			if err != nil {
				return fmt.Errorf("could not decrypt master key: %w", err)
			}
			encryptedMasterKey = decryptedMasterKey
		}
		return nil
	})
	s.log.Info().Str("Login", login).Bool("hasToken", token != "").Msg("Retrieved user credentials")
	return login, token, device, encryptedMasterKey, err
}
