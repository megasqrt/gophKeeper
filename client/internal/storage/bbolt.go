package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/scrypt"
)

var (
	// ErrUserNotRegistered означает, что данные пользователя не найдены в локальном хранилище.
	ErrUserNotRegistered = errors.New("user not registered or logged in")
)

// Определим имена "бакетов" (buckets)
var (
	configBucket    = []byte("Config")
	passwordsBucket = []byte("Passwords")
	cardsBucket     = []byte("CreditCards")
	textBucket      = []byte("TextData")
	binaryBucket    = []byte("BinaryData")
)

// Определим ключи для хранения конфигурации.
var (
	loginKey    = []byte("login")
	userKey     = []byte("user")
	passwordKey = []byte("password")
	tokenKey    = []byte("token")
	lastSyncKey = []byte("lastSync")
)

// BboltStorage представляет собой хранилище на базе bbolt.
type BboltStorage struct {
	db   *bbolt.DB
	key  []byte // Ключ шифрования, активен в течение сессии
	log  zerolog.Logger
	user string
}

// NewBboltStorage создает и инициализирует новое хранилище bbolt.
func NewBboltStorage(path string, log zerolog.Logger) (*BboltStorage, error) {
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})

	if err != nil {
		return nil, fmt.Errorf("could not open db: %w", err)
	}

	// Создаем бакеты, если они еще не существуют.
	err = db.Update(func(tx *bbolt.Tx) error {
		buckets := [][]byte{
			configBucket,
			passwordsBucket,
			cardsBucket,
			textBucket,
			binaryBucket,
		}
		for _, bucket := range buckets {
			if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
				return fmt.Errorf("could not create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not set up buckets: %w", err)
	}

	return &BboltStorage{db: db, log: log}, nil
}

// Unlock генерирует ключ шифрования из пароля и сохраняет его в сессии.
func (s *BboltStorage) Unlock(user, password string) error {
	s.log.Info().Str("user", user).Msg("Deriving encryption key from password")
	// Используем имя пользователя как "соль" для scrypt. Это не идеально, но просто.
	key, err := encriptPassword(user, password)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to derive key")
		return fmt.Errorf("could not derive key: %w", err)
	}
	s.key = key

	// Проверяем ключ, пытаясь расшифровать проверочное значение
	err = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		userBytes := b.Get(userKey)
		if userBytes == nil {
			s.log.Error().Msg("User not found in storage")
			return errors.New("user not found in storage")
		}
		unlockUser, err := s.decrypt(userBytes)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to decrypt user")
			return err
		}
		if string(unlockUser) != user {
			return errors.New("invalid password")
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.log.Info().Msg("Storage unlocked successfully")
	return nil
}

func encriptPassword(user, password string) ([]byte, error) {
	salt := []byte(user)
	// N=32768, r=8, p=1 - стандартные параметры для интерактивного входа
	key, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// IsLoggedIn проверяет, сохранен ли токен.
func (s *BboltStorage) IsLoggedIn() bool {
	_, token, err := s.GetUserCredentials()
	return err == nil && token != ""
}

func (s *BboltStorage) IsFirstRun() bool {
	s.log.Info().Msg("Checking if first run")
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		passwordBytes := b.Get(passwordKey)
		if len(passwordBytes) > 0 {
			return nil
		}
		return errors.New("password not found in storage")
	})
	if err != nil {
		s.log.Info().Msg("Password not found in storage")
		return true
	}
	s.log.Info().Msg("Password found in storage")
	return false
}

// Close закрывает соединение с базой данных.
func (s *BboltStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *BboltStorage) encrypt(data []byte) ([]byte, error) {
	if s.key == nil {
		return nil, errors.New("storage is locked, cannot encrypt")
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (s *BboltStorage) decrypt(data []byte) ([]byte, error) {
	if s.key == nil {
		return nil, errors.New("storage is locked, cannot decrypt")
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
