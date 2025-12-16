package sqlite

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	models "gophKeeper/pkg/grpchelper"

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
	// Используем имя пользователя как "соль" для scrypt. Это не идеально, но просто.
	key, err := encriptPassword(user, password)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to derive key")
		return fmt.Errorf("could not derive key: %w", err)
	}
	s.key = key
	s.user = user

	// Проверяем ключ, пытаясь расшифровать проверочное значение
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
	// N=32768, r=8, p=1 - стандартные параметры для интерактивного входа
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
		// Логируем ошибку для диагностики
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

	// Производный ключ для шифрования
	key, err := encriptPassword(user, password)
	if err != nil {
		return fmt.Errorf("could not derive key: %w", err)
	}
	s.key = key // Устанавливаем ключ для сессии
	s.user = user

	// Шифруем пароль для проверки при Unlock
	encryptedPassword, err := s.encrypt([]byte(password))
	if err != nil {
		return fmt.Errorf("could not encrypt password for verification: %w", err)
	}

	// Начинаем транзакцию
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Откатываем в случае ошибки

	// Сохраняем зашифрованный пароль (user используется как PRIMARY KEY)
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

	// Обновляем все поля в таблице config
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

	// Расшифровываем токен и device ID
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

// calculateCardChecksum вычисляет checksum для карты
func calculateCardChecksum(number, holder, expiry, cvv string) string {
	h := sha256.New()
	h.Write([]byte(number))
	h.Write([]byte(holder))
	h.Write([]byte(expiry))
	h.Write([]byte(cvv))
	return hex.EncodeToString(h.Sum(nil))
}

// calculateTextChecksum вычисляет checksum для текста
func calculateTextChecksum(title, text string) string {
	h := sha256.New()
	h.Write([]byte(title))
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

// calculateFileChecksum вычисляет checksum для файла
func calculateFileChecksum(name string, size int64, metadata string) string {
	h := sha256.New()
	h.Write([]byte(name))
	h.Write([]byte(fmt.Sprintf("%d", size)))
	h.Write([]byte(metadata))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *SqliteStorage) SaveCard(cardData *models.Card) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	cardData.ChangeTime = now
	cardData.Checksum = calculateCardChecksum(cardData.Number, cardData.Holder, cardData.Expiry, cardData.CVV)

	// Шифруем чувствительные данные
	encryptedNumber, err := s.encryptString(cardData.Number)
	if err != nil {
		return fmt.Errorf("could not encrypt card number: %w", err)
	}
	encryptedHolder, err := s.encryptString(cardData.Holder)
	if err != nil {
		return fmt.Errorf("could not encrypt card holder: %w", err)
	}
	encryptedExpiry, err := s.encryptString(cardData.Expiry)
	if err != nil {
		return fmt.Errorf("could not encrypt expiry date: %w", err)
	}
	encryptedCVC, err := s.encryptString(cardData.CVV)
	if err != nil {
		return fmt.Errorf("could not encrypt CVC: %w", err)
	}

	var encryptedMetadata []byte
	if cardData.Metadata != "" {
		encryptedMetadata, err = s.encryptString(cardData.Metadata)
		if err != nil {
			return fmt.Errorf("could not encrypt metadata: %w", err)
		}
	}

	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO cards 
		(id, card_number_data, card_holder_data, expiry_date_data, cvc_data, metadata, checksum, created_at, updated_at, deleted_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cardData.LocalID, encryptedNumber, encryptedHolder, encryptedExpiry, encryptedCVC,
		encryptedMetadata, cardData.Checksum, now, now,
		sql.NullInt64{Valid: cardData.Deleted, Int64: now})
	return err
}

func (s *SqliteStorage) UpdateCard(cardData *models.Card) error {
	return s.SaveCard(cardData) // Используем тот же метод
}

func (s *SqliteStorage) GetCards() ([]models.Card, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, card_number_data, card_holder_data, expiry_date_data, cvc_data, metadata, checksum, created_at, updated_at, deleted_at 
		FROM cards 
		WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []models.Card
	for rows.Next() {
		var c models.Card
		var encryptedNumber, encryptedHolder, encryptedExpiry, encryptedCVC, encryptedMetadata []byte
		var deletedAt sql.NullInt64
		var createdAt int64

		if err := rows.Scan(&c.LocalID, &encryptedNumber, &encryptedHolder, &encryptedExpiry, &encryptedCVC,
			&encryptedMetadata, &c.Checksum, &createdAt, &c.ChangeTime, &deletedAt); err != nil {
			s.log.Error().Err(err).Msg("Failed to scan card")
			continue
		}

		// Расшифровываем данные
		c.Number, err = s.decryptString(encryptedNumber)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to decrypt card number")
			continue
		}
		c.Holder, err = s.decryptString(encryptedHolder)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to decrypt card holder")
			continue
		}
		c.Expiry, err = s.decryptString(encryptedExpiry)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to decrypt expiry date")
			continue
		}
		c.CVV, err = s.decryptString(encryptedCVC)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to decrypt CVC")
			continue
		}
		if len(encryptedMetadata) > 0 {
			c.Metadata, err = s.decryptString(encryptedMetadata)
			if err != nil {
				s.log.Error().Err(err).Msg("Failed to decrypt metadata")
				continue
			}
		}
		c.Deleted = deletedAt.Valid

		cards = append(cards, c)
	}
	return cards, rows.Err()
}

func (s *SqliteStorage) GetCardsByIDs(ids []int64) ([]models.Card, error) {
	// Not implemented yet
	return nil, nil
}

func (s *SqliteStorage) DeleteCard(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE cards SET deleted_at = ? WHERE id = ?", now, id)
	return err
}

func (s *SqliteStorage) SaveLastSyncTime(t int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Получаем текущего пользователя
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

func (s *SqliteStorage) SaveText(textData *models.TextData) error {
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
		(id, title, text, checksum, created_at, updated_at, deleted_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		textData.LocalID, encryptedTitle, encryptedText, textData.Checksum,
		now, now, sql.NullInt64{Valid: textData.Deleted, Int64: now})
	return err
}

func (s *SqliteStorage) UpdateText(textData *models.TextData) error {
	return s.SaveText(textData) // Используем тот же метод
}

func (s *SqliteStorage) GetTexts() ([]models.TextData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, title, text, checksum, created_at, updated_at, deleted_at 
		FROM note 
		WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var texts []models.TextData
	for rows.Next() {
		var t models.TextData
		var encryptedTitle, encryptedText []byte
		var deletedAt sql.NullInt64
		var createdAt int64

		if err := rows.Scan(&t.LocalID, &encryptedTitle, &encryptedText, &t.Checksum,
			&createdAt, &t.ChangeTime, &deletedAt); err != nil {
			s.log.Error().Err(err).Msg("Failed to scan text")
			continue
		}

		// Расшифровываем данные
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

func (s *SqliteStorage) GetTextsByIDs(ids []int64) ([]models.TextData, error) {
	// Not implemented yet
	return nil, nil
}

func (s *SqliteStorage) DeleteText(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE note SET deleted_at = ? WHERE id = ?", now, id)
	return err
}

func (s *SqliteStorage) GetFiles() ([]models.FileData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, metadata, size, checksum, created_at, updated_at, deleted_at 
		FROM binary_data 
		WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []models.FileData
	for rows.Next() {
		var f models.FileData
		var encryptedName, encryptedMetadata []byte
		var deletedAt sql.NullInt64
		var createdAt int64

		if err := rows.Scan(&f.LocalID, &encryptedName, &encryptedMetadata, &f.Size,
			&f.Checksum, &createdAt, &f.ChangeTime, &deletedAt); err != nil {
			s.log.Error().Err(err).Msg("Failed to scan file")
			continue
		}

		// Расшифровываем данные
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

func (s *SqliteStorage) GetFilesByIDs(ids []int64) ([]models.FileData, error) {
	// Not implemented yet
	return nil, nil
}

func (s *SqliteStorage) GetFileByID(id int64) (map[string]interface{}, error) {
	// Not implemented yet
	return nil, nil
}

func (s *SqliteStorage) UpdateFile(data *models.FileData) error {
	return s.SaveFileMetadata(data)
}

func (s *SqliteStorage) SaveFile(data *models.FileData, content []byte) error {
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

	// Шифруем метаданные
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

	// Шифруем содержимое файла
	encryptedContent, err := s.encrypt(content)
	if err != nil {
		return fmt.Errorf("could not encrypt file content: %w", err)
	}

	// Сохраняем в binary_data (data = содержимое файла)
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO binary_data 
		(id, data, name, metadata, size, checksum, created_at, updated_at, deleted_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		data.LocalID, encryptedContent, encryptedName, encryptedMetadata,
		data.Size, data.Checksum, now, now,
		sql.NullInt64{Valid: data.Deleted, Int64: now})
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SqliteStorage) SaveFileMetadata(data *models.FileData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	data.ChangeTime = now
	data.Checksum = calculateFileChecksum(data.Name, data.Size, data.Metadata)

	// Шифруем метаданные
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

	// Обновляем только метаданные, содержимое не трогаем
	_, err = s.db.Exec(`
		UPDATE binary_data 
		SET name = ?, metadata = ?, size = ?, checksum = ?, updated_at = ? 
		WHERE id = ?`,
		encryptedName, encryptedMetadata, data.Size, data.Checksum, now, data.LocalID)
	return err
}

func (s *SqliteStorage) DeleteFileByID(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	_, err := s.db.Exec("UPDATE binary_data SET deleted_at = ? WHERE id = ?", now, id)
	return err
}

// Close закрывает соединение с базой данных.
func (s *SqliteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
