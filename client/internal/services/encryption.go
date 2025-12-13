package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	model "gophKeeper/pkg/grpchelper"
	"golang.org/x/crypto/scrypt"
)

// EncryptionService управляет шифрованием данных на клиенте
// Реализует интерфейс grpchelper.Encryptor
type EncryptionService struct {
	masterKey []byte // Мастер-ключ пользователя
}

// Глобальный экземпляр EncryptionService для использования в конвертерах
var globalEncryptionService *EncryptionService

// NewEncryptionService создает новый сервис шифрования
func NewEncryptionService() *EncryptionService {
	return &EncryptionService{}
}

// SetGlobalEncryptionService устанавливает глобальный экземпляр EncryptionService
// и регистрирует его в grpchelper для использования в конвертерах
func SetGlobalEncryptionService(service *EncryptionService) {
	globalEncryptionService = service
	// Регистрируем сервис в grpchelper для использования в конвертерах
	model.SetGlobalEncryptor(service)
}

// SetMasterKey устанавливает мастер-ключ (расшифрованный)
func (s *EncryptionService) SetMasterKey(masterKey []byte) {
	s.masterKey = masterKey
}

// HasMasterKey проверяет, установлен ли мастер-ключ
func (s *EncryptionService) HasMasterKey() bool {
	return len(s.masterKey) > 0
}

// DecryptMasterKey расшифровывает мастер-ключ из зашифрованного значения паролем
func (s *EncryptionService) DecryptMasterKey(encryptedMasterKey []byte, password string, salt []byte) error {
	// Получаем ключ из пароля через scrypt
	derivedKey, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return fmt.Errorf("failed to derive key from password: %w", err)
	}

	// Расшифровываем мастер-ключ
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedMasterKey) < nonceSize {
		return errors.New("encrypted master key too short")
	}

	nonce, ciphertext := encryptedMasterKey[:nonceSize], encryptedMasterKey[nonceSize:]
	masterKey, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt master key: %w", err)
	}

	s.masterKey = masterKey
	return nil
}

// EncryptString шифрует строку мастер-ключом и возвращает base64 строку
func (s *EncryptionService) EncryptString(data string) (string, error) {
	if !s.HasMasterKey() {
		return "", errors.New("master key not set")
	}

	encrypted, err := s.encryptData([]byte(data))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString расшифровывает base64 строку мастер-ключом
func (s *EncryptionService) DecryptString(encryptedBase64 string) (string, error) {
	if !s.HasMasterKey() {
		return "", errors.New("master key not set")
	}

	encrypted, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := s.decryptData(encrypted)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// encryptData шифрует данные мастер-ключом
func (s *EncryptionService) encryptData(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
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

// decryptData расшифровывает данные мастер-ключом
func (s *EncryptionService) decryptData(encryptedData []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
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

