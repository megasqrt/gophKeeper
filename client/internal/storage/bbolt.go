package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"gophKeeper/client/internal/domain"
	"gophKeeper/pkg/grpchelper"
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
	fileMetaBucket  = []byte("FileMetadata") // Бакет для метаданных файлов
	fileDataBucket  = []byte("FileData")     // Бакет для содержимого файлов
)

// Определим ключи для хранения конфигурации.
var (
	loginKey    = []byte("login")
	userKey     = []byte("user")
	passwordKey = []byte("password")
	tokenKey    = []byte("token")
	deviceKey   = []byte("device")
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
			fileMetaBucket,
			fileDataBucket,
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
	_, token, _, _, err := s.GetUserCredentials()
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

// saveItem — это универсальный метод для сохранения или обновления элемента в бакете.
// Если isNew=true, генерируется новый ID. В качестве ключа используется ID.
func (s *BboltStorage) saveItem(bucketName []byte, item domain.Syncable, isNew bool) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return s.saveItemTx(tx, bucketName, item, isNew)
	})
}

// saveItemTx — внутренняя версия saveItem, работающая внутри существующей транзакции.
func (s *BboltStorage) saveItemTx(tx *bbolt.Tx, bucketName []byte, item domain.Syncable, isNew bool) error {
	b := tx.Bucket(bucketName)
	var itemID string

	if isNew {
		id, _ := b.NextSequence()
		itemID = fmt.Sprintf("%d", id)
		// Устанавливаем ID в модели через интерфейс
		if settable, ok := item.(interface{ SetLocalID(string) }); ok {
			settable.SetLocalID(itemID)
		} else {
			return fmt.Errorf("item in bucket %s does not support setting local ID", bucketName)
		}
	} else {
		itemID = item.GetLocalID()
		if itemID == "" {
			return fmt.Errorf("item ID is missing for update in bucket %s", bucketName)
		}
	}

	encryptedData, err := s.encryptItem(item)
	if err != nil {
		return fmt.Errorf("could not encrypt item data for bucket %s: %w", bucketName, err)
	}

	return b.Put([]byte(itemID), encryptedData)
}

// getAllItems — это универсальный метод для получения всех элементов из бакета.
func (s *BboltStorage) getAllItems(bucketName []byte, newItemFunc func() interface{}) ([]interface{}, error) {
	var items []interface{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.ForEach(func(k, v []byte) error {
			// Создаем новый пустой экземпляр модели
			item := newItemFunc()

			// Десериализуем напрямую в него
			err := s.decryptItem(v, item)
			if err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msgf("Could not decrypt or unmarshal item in bucket %s", bucketName)
				// Пропускаем записи, которые не удалось сконвертировать
				return nil
			}

			items = append(items, item)
			return nil
		})
	})

	if err != nil {
		s.log.Error().Err(err).Msgf("Failed to get all items from bucket %s", bucketName)
	}
	return items, err
}

// GetAllSyncInfo извлекает только ключевые поля для синхронизации (LocalID, ServerID, Checksum)
// из всех элементов в бакете, избегая полной дешифрации и десериализации.
func (s *BboltStorage) GetShortSyncInfo(bucketName []byte) ([]grpchelper.SyncInfo, error) {
	var syncInfos []grpchelper.SyncInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.ForEach(func(k, v []byte) error {
			var info grpchelper.SyncInfo

			// Десериализуем только необходимые поля
			err := s.decryptItem(v, &info)
			if err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msgf("Could not decrypt or unmarshal sync info in bucket %s", bucketName)
				// Пропускаем записи, которые не удалось обработать
				return nil
			}

			syncInfos = append(syncInfos, info)
			return nil
		})
	})

	if err != nil {
		s.log.Error().Err(err).Msgf("Failed to get all sync info from bucket %s", bucketName)
	}
	return syncInfos, err
}

// markAsDeleted помечает элемент как удаленный, вместо физического удаления.
func (s *BboltStorage) markAsDeleted(bucketName []byte, id string, newItemFunc func() interface{}) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		encryptedData := b.Get([]byte(id))
		if encryptedData == nil {
			return fmt.Errorf("item with id '%s' not found in bucket %s", id, bucketName)
		}

		item := newItemFunc()
		if err := s.decryptItem(encryptedData, item); err != nil {
			return fmt.Errorf("could not decrypt item for deletion: %w", err)
		}

		// Устанавливаем флаг Deleted через интерфейс, если он поддерживается
		if deletable, ok := item.(interface{ SetDeleted(bool) }); ok {
			deletable.SetDeleted(true)
		} else {
			return fmt.Errorf("item does not support soft deletion")
		}

		return s.saveItem(bucketName, item.(domain.Syncable), false)
	})
}

// getItemByID — это универсальный метод для получения одного элемента по ID.
func (s *BboltStorage) getItemByID(bucketName []byte, id string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		itemData, err := s.getItemByIDTx(b, id)
		if err != nil {
			return err
		}
		result = itemData
		return nil
	})
	return result, err
}

// getItemByIDTx — внутренняя версия, работающая с объектом бакета внутри транзакции.
func (s *BboltStorage) getItemByIDTx(b *bbolt.Bucket, id string) (map[string]interface{}, error) {
	encryptedData := b.Get([]byte(id))
	if encryptedData == nil {
		return nil, fmt.Errorf("item with id '%s' not found in bucket", id)
	}
	var itemData map[string]interface{}
	err := s.decryptItem(encryptedData, &itemData)
	return itemData, err
}

// deleteItem — это универсальный метод для удаления элемента из бакета по ID.
func (s *BboltStorage) deleteItem(bucketName []byte, id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if err := b.Delete([]byte(id)); err != nil {
			return fmt.Errorf("could not delete item with id '%s' from bucket %s: %w", id, bucketName, err)
		}
		return nil
	})
}

// encryptItem сериализует и шифрует любой объект.
func (s *BboltStorage) encryptItem(item interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("could not marshal item: %w", err)
	}
	return s.encrypt(jsonData)
}

// decryptItem дешифрует и десериализует данные в предоставленный объект.
func (s *BboltStorage) decryptItem(data []byte, target interface{}) error {
	decryptedData, err := s.decrypt(data)
	if err != nil {
		return fmt.Errorf("could not decrypt item: %w", err)
	}
	if err := json.Unmarshal(decryptedData, target); err != nil {
		return fmt.Errorf("could not unmarshal item: %w", err)
	}
	return nil
}
