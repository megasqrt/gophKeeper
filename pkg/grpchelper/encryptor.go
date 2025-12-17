package grpchelper

// Encryptor интерфейс для шифрования строк
type Encryptor interface {
	EncryptString(data string) (string, error)
	DecryptString(encryptedBase64 string) (string, error)
}

// Глобальный экземпляр Encryptor для использования в конвертерах
var globalEncryptor Encryptor

// SetGlobalEncryptor устанавливает глобальный экземпляр Encryptor
func SetGlobalEncryptor(encryptor Encryptor) {
	globalEncryptor = encryptor
}

// GetGlobalEncryptor возвращает глобальный экземпляр Encryptor
func GetGlobalEncryptor() Encryptor {
	return globalEncryptor
}
