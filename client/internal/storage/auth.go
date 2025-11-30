package storage

import (
	"fmt"

	"go.etcd.io/bbolt"
)

// SaveUserCredentials сохраняет пароль пользователя.
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

// SaveUserCredentials сохраняет токен пользователя.
func (s *BboltStorage) SaveUserCredentials(login, token string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		encryptedToken, err := s.encrypt([]byte(token))
		if err != nil {
			return fmt.Errorf("could not encrypt token: %w", err)
		}
		return b.Put(tokenKey, encryptedToken)
	})
}

// GetUserCredentials извлекает токен пользователя.
func (s *BboltStorage) GetUserCredentials() (user,token string, err error) {
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
		userBytes := b.Get(userKey)
		if len(userBytes) > 0 {
			decryptedLogin, err := s.decrypt(userBytes)	
			if err!= nil {
				return fmt.Errorf("could not decrypt login: %w", err)
			}
			user = string(decryptedLogin)
		}
		return nil
	})
	s.log.Info().Str("user", user).Str("token", token).Msg("Retrieved user credentials")
	return user,token, nil
}