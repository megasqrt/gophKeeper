package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

// MasterKeyService управляет мастер-ключами пользователей
type MasterKeyService struct{}

// NewMasterKeyService создает новый сервис для работы с мастер-ключами
func NewMasterKeyService() *MasterKeyService {
	return &MasterKeyService{}
}

// GenerateMasterKey генерирует новый мастер-ключ для пользователя (32 байта для AES-256)
func (s *MasterKeyService) GenerateMasterKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256 требует 32 байта
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}
	return key, nil
}

// EncryptMasterKeyWithPassword шифрует мастер-ключ паролем пользователя
// Использует scrypt для получения ключа из пароля, затем AES-GCM для шифрования
func (s *MasterKeyService) EncryptMasterKeyWithPassword(masterKey []byte, password string, salt []byte) ([]byte, error) {
	// Получаем ключ из пароля через scrypt
	derivedKey, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key from password: %w", err)
	}

	// Шифруем мастер-ключ используя AES-GCM
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := gcm.Seal(nonce, nonce, masterKey, nil)
	return encrypted, nil
}

// DecryptMasterKeyWithPassword расшифровывает мастер-ключ паролем пользователя
func (s *MasterKeyService) DecryptMasterKeyWithPassword(encryptedMasterKey []byte, password string, salt []byte) ([]byte, error) {
	// Получаем ключ из пароля через scrypt
	derivedKey, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key from password: %w", err)
	}

	// Расшифровываем мастер-ключ
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedMasterKey) < nonceSize {
		return nil, errors.New("encrypted master key too short")
	}

	nonce, ciphertext := encryptedMasterKey[:nonceSize], encryptedMasterKey[nonceSize:]
	masterKey, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt master key: %w", err)
	}

	return masterKey, nil
}

// EncryptData шифрует данные мастер-ключом
func (s *MasterKeyService) EncryptData(data []byte, masterKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := gcm.Seal(nonce, nonce, data, nil)
	return encrypted, nil
}

// DecryptData расшифровывает данные мастер-ключом
func (s *MasterKeyService) DecryptData(encryptedData []byte, masterKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, errors.New("encrypted data too short")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	data, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return data, nil
}

