package sqlite

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/scrypt"

	_ "github.com/mattn/go-sqlite3"
)

var (
	// ErrUserNotRegistered означает, что данные пользователя не найдены в локальном хранилище.
	ErrUserNotRegistered = errors.New("user not registered or logged in")
)

// SqliteStorage представляет собой хранилище на базе SQLite.
type SqliteStorage struct {
	db   *sql.DB
	key  []byte // Ключ шифрования, активен в течение сессии
	log  zerolog.Logger
	user string
	mu   sync.RWMutex
}

// NewSqliteStorage создает и инициализирует новое хранилище SQLite.
func NewSqliteStorage(path string, log zerolog.Logger) (*SqliteStorage, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_journal_mode=WAL", path))
	if err != nil {
		return nil, fmt.Errorf("could not open db: %w", err)
	}

	db.SetMaxOpenConns(1)

	return &SqliteStorage{db: db, log: log}, nil
}

func (s *SqliteStorage) GetDB() *sql.DB {
	return s.db
}

// Unlock генерирует ключ шифрования из пароля и сохраняет его в сессии.
func (s *SqliteStorage) Unlock(user, password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.log.Info().Str("user", user).Msg("Deriving encryption key from password")
	key, err := encriptPassword(user, password)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to derive key")
		return fmt.Errorf("could not derive key: %w", err)
	}
	s.key = key
	s.user = user

	var passwordBytes []byte
	err = s.db.QueryRow("SELECT password_hash FROM config WHERE user = ?", user).Scan(&passwordBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			s.log.Error().Msg("User not found in storage")
			return errors.New("user not found in storage")
		}
		return err
	}

	decryptedPassword, err := s.decrypt(passwordBytes)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to decrypt password")
		return err
	}
	if string(decryptedPassword) != password {
		return errors.New("invalid password")
	}

	s.log.Info().Msg("Storage unlocked successfully")
	return nil
}

func encriptPassword(user, password string) ([]byte, error) {
	salt := []byte(user)
	key, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (s *SqliteStorage) encrypt(data []byte) ([]byte, error) {
	if s.key == nil {
		s.log.Error().Msg("Attempted to encrypt with nil key - storage is locked")
		return nil, errors.New("storage is locked, cannot encrypt")
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to create cipher for encryption")
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to create GCM for encryption")
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		s.log.Error().Err(err).Msg("Failed to generate nonce for encryption")
		return nil, err
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (s *SqliteStorage) decrypt(data []byte) ([]byte, error) {
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
	decrypted, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		s.log.Error().Err(err).Msg("Decryption failed - key may be incorrect or data corrupted")
		return nil, err
	}
	return decrypted, nil
}

// encryptString шифрует строку и возвращает зашифрованные байты
func (s *SqliteStorage) encryptString(data string) ([]byte, error) {
	return s.encrypt([]byte(data))
}

// decryptString расшифровывает байты и возвращает строку
func (s *SqliteStorage) decryptString(data []byte) (string, error) {
	decrypted, err := s.decrypt(data)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}



func (s *SqliteStorage) SaveLastSyncTime(t int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var user string
	err := s.db.QueryRow("SELECT user FROM config WHERE login IS NOT NULL LIMIT 1").Scan(&user)
	if err != nil {
		return fmt.Errorf("could not get user: %w", err)
	}

	_, err = s.db.Exec("UPDATE config SET last_sync = ? WHERE user = ?", t, user)
	return err
}

func (s *SqliteStorage) GetLastSyncTime() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var lastSync sql.NullInt64
	err := s.db.QueryRow("SELECT last_sync FROM config WHERE login IS NOT NULL LIMIT 1").Scan(&lastSync)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if !lastSync.Valid {
		return 0, nil
	}
	return lastSync.Int64, nil
}




// Close закрывает соединение с базой данных.
func (s *SqliteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
