package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"go.etcd.io/bbolt"
	"golang.org/x/crypto/scrypt"
	"github.com/rs/zerolog"
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
	tokenKey    = []byte("token")
	lastSyncKey = []byte("lastSync")
)

// BboltStorage представляет собой хранилище на базе bbolt.
type BboltStorage struct {
	db  *bbolt.DB
	key []byte // Ключ шифрования, активен в течение сессии
	log zerolog.Logger
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

	return &BboltStorage{db: db}, nil
}

// Unlock генерирует ключ шифрования из пароля и сохраняет его в сессии.
func (s *BboltStorage) Unlock(login, password string) error {
	s.log.Info().Str("login", login).Msg("Deriving encryption key from password")
	// Используем логин как "соль" для scrypt. Это не идеально, но просто.
	salt := []byte(login)
	// N=32768, r=8, p=1 - стандартные параметры для интерактивного входа
	key, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to derive key")
		return fmt.Errorf("could not derive key: %w", err)
	}
	s.key = key
	s.log.Info().Msg("Storage unlocked successfully")
	return nil
}

// SaveUserCredentials сохраняет логин и токен пользователя.
func (s *BboltStorage) SaveUserCredentials(login, token string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		// Логин сохраняем в открытом виде, он нужен для восстановления "соли"
		if err := b.Put(loginKey, []byte(login)); err != nil {
			return err
		}
		encryptedToken, err := s.encrypt([]byte(token))
		if err != nil {
			return fmt.Errorf("could not encrypt token: %w", err)
		}
		return b.Put(tokenKey, encryptedToken)
	})
}

// GetUserCredentials извлекает логин и токен пользователя.
func (s *BboltStorage) GetUserCredentials() (login, token string, err error) {
	err = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)

		loginBytes := b.Get(loginKey)
		if loginBytes == nil {
			// Если логин не найден, считаем, что пользователь не зарегистрирован.
			return ErrUserNotRegistered
		}
		login = string(loginBytes)

		tokenBytes := b.Get(tokenKey)
		if len(tokenBytes) > 0 {
			decryptedToken, err := s.decrypt(tokenBytes)
			if err != nil {
				// Если не можем расшифровать, возможно, пароль неверный
				return fmt.Errorf("could not decrypt token: %w", err)
			}
			token = string(decryptedToken)
		}

		return nil
	})
	return login, token, err
}

// IsLoggedIn проверяет, сохранен ли токен.
func (s *BboltStorage) IsLoggedIn() bool {
	_, token, err := s.GetUserCredentials()
	return err == nil && token != ""
}

// SaveLastSyncTime сохраняет время последней успешной синхронизации с сервером.
func (s *BboltStorage) SaveLastSyncTime(t time.Time) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		timeBytes, err := t.MarshalText() // MarshalText более устойчив к изменениям
		if err != nil {
			return fmt.Errorf("could not marshal time: %w", err)
		}
		encryptedTime, err := s.encrypt(timeBytes)
		if err != nil {
			return fmt.Errorf("could not encrypt sync time: %w", err)
		}
		return b.Put(lastSyncKey, encryptedTime)
	})
}

// GetLastSyncTime извлекает время последней успешной синхронизации.
func (s *BboltStorage) GetLastSyncTime() (time.Time, error) {
	var t time.Time
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(configBucket)
		timeBytes := b.Get(lastSyncKey)
		if timeBytes == nil {
			return nil // Время еще не сохранено
		}
		decryptedTime, err := s.decrypt(timeBytes)
		if err != nil {
			return fmt.Errorf("could not decrypt sync time: %w", err)
		}
		return t.UnmarshalText(decryptedTime)
	})
	return t, err
}

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
