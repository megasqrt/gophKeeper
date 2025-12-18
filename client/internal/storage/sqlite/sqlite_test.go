package sqlite

import (
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*SqliteStorage, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := zerolog.Nop()

	storage, err := NewSqliteStorage(dbPath, log)
	require.NoError(t, err)

	// Применяем миграции напрямую через SQL
	_, err = storage.db.Exec(`
		CREATE TABLE IF NOT EXISTS config (
			login TEXT,    
			user TEXT PRIMARY KEY NOT NULL,     
			password_hash TEXT NOT NULL,
			token TEXT,
			device TEXT,
			last_sync INTEGER,
			encrypted_master_key BLOB
		);
		
		CREATE TABLE IF NOT EXISTS credentials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id TEXT,
			data BLOB NOT NULL,
			checksum TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER,
			version INTEGER
		);
	`)
	require.NoError(t, err)

	return storage, dbPath
}

func TestNewSqliteStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	log := zerolog.Nop()

	storage, err := NewSqliteStorage(dbPath, log)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.NotNil(t, storage.GetDB())

	err = storage.Close()
	assert.NoError(t, err)
}

func TestLocalRegister(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	err := storage.LocalRegister(user, password)
	require.NoError(t, err)

	// Проверяем, что пользователь зарегистрирован
	isFirstRun := storage.IsFirstRun()
	assert.False(t, isFirstRun)
}

func TestUnlock_Success(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// Сначала регистрируем пользователя
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)

	// Теперь разблокируем
	err = storage.Unlock(user, password)
	assert.NoError(t, err)
}

func TestUnlock_InvalidPassword(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"
	wrongPassword := "wrongpassword"

	// Регистрируем пользователя
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)

	// Пытаемся разблокировать с неправильным паролем
	err = storage.Unlock(user, wrongPassword)
	assert.Error(t, err)
	// Ошибка может быть "invalid password" или ошибка расшифровки
	assert.True(t, err.Error() == "invalid password" || err.Error() == "cipher: message authentication failed")
}

func TestUnlock_UserNotFound(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "nonexistent"
	password := "testpassword"

	err := storage.Unlock(user, password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestSaveUserCredentials(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// Регистрируем и разблокируем
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)
	err = storage.Unlock(user, password)
	require.NoError(t, err)

	// Сохраняем учетные данные
	login := "testuser"
	token := "test-token-123"
	deviceID := "device-id-456"
	encryptedMasterKey := []byte("encrypted-key")

	err = storage.SaveUserCredentials(login, token, deviceID, encryptedMasterKey)
	assert.NoError(t, err)
}

func TestGetUserCredentials_Success(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// Регистрируем и разблокируем
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)
	err = storage.Unlock(user, password)
	require.NoError(t, err)

	// Сохраняем учетные данные
	login := "testuser"
	token := "test-token-123"
	deviceID := "device-id-456"
	encryptedMasterKey := []byte("encrypted-key")

	err = storage.SaveUserCredentials(login, token, deviceID, encryptedMasterKey)
	require.NoError(t, err)

	// Получаем учетные данные
	retrievedLogin, retrievedToken, retrievedDeviceID, retrievedMasterKey, err := storage.GetUserCredentials()
	assert.NoError(t, err)
	assert.Equal(t, login, retrievedLogin)
	assert.Equal(t, token, retrievedToken)
	assert.Equal(t, deviceID, retrievedDeviceID)
	assert.Equal(t, encryptedMasterKey, retrievedMasterKey)
}

func TestGetUserCredentials_StorageLocked(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	// Пытаемся получить учетные данные без разблокировки
	_, _, _, _, err := storage.GetUserCredentials()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage is locked")
}

func TestGetUserCredentials_UserNotRegistered(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// Разблокируем, но не регистрируем
	err := storage.Unlock(user, password)
	// Это должно вернуть ошибку, так как пользователь не найден
	assert.Error(t, err)
}

func TestIsLoggedIn(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// До регистрации
	isLoggedIn := storage.IsLoggedIn()
	assert.False(t, isLoggedIn)

	// Регистрируем и разблокируем
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)
	err = storage.Unlock(user, password)
	require.NoError(t, err)

	// Сохраняем учетные данные
	login := "testuser"
	token := "test-token-123"
	deviceID := "device-id-456"
	encryptedMasterKey := []byte("encrypted-key")

	err = storage.SaveUserCredentials(login, token, deviceID, encryptedMasterKey)
	require.NoError(t, err)

	// После сохранения учетных данных
	isLoggedIn = storage.IsLoggedIn()
	assert.True(t, isLoggedIn)
}

func TestIsFirstRun(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	// При первом запуске
	isFirstRun := storage.IsFirstRun()
	assert.True(t, isFirstRun)

	// После регистрации
	user := "testuser"
	password := "testpassword"
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)

	isFirstRun = storage.IsFirstRun()
	assert.False(t, isFirstRun)
}

func TestEncryptDecrypt(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// Регистрируем и разблокируем
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)
	err = storage.Unlock(user, password)
	require.NoError(t, err)

	// Тестируем шифрование/расшифровку
	originalData := []byte("test data to encrypt")

	encrypted, err := storage.encrypt(originalData)
	require.NoError(t, err)
	assert.NotNil(t, encrypted)
	assert.NotEqual(t, originalData, encrypted)

	decrypted, err := storage.decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, originalData, decrypted)
}

func TestEncrypt_StorageLocked(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	// Пытаемся зашифровать без разблокировки
	data := []byte("test data")
	_, err := storage.encrypt(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage is locked")
}

func TestDecrypt_StorageLocked(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	// Пытаемся расшифровать без разблокировки
	data := []byte("encrypted data")
	_, err := storage.decrypt(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage is locked")
}

func TestEncryptString_DecryptString(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// Регистрируем и разблокируем
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)
	err = storage.Unlock(user, password)
	require.NoError(t, err)

	// Тестируем шифрование/расшифровку строк
	originalString := "test string to encrypt"

	encrypted, err := storage.encryptString(originalString)
	require.NoError(t, err)
	assert.NotNil(t, encrypted)

	decrypted, err := storage.decryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, originalString, decrypted)
}

func TestSaveLastSyncTime_GetLastSyncTime(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// Регистрируем и разблокируем
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)
	err = storage.Unlock(user, password)
	require.NoError(t, err)

	// Сохраняем время синхронизации
	syncTime := int64(1234567890)
	err = storage.SaveLastSyncTime(syncTime)
	require.NoError(t, err)

	// Получаем время синхронизации
	retrievedTime, err := storage.GetLastSyncTime()
	require.NoError(t, err)
	assert.Equal(t, syncTime, retrievedTime)
}

func TestGetLastSyncTime_NoSyncTime(t *testing.T) {
	storage, _ := setupTestDB(t)
	defer storage.Close()

	user := "testuser"
	password := "testpassword"

	// Регистрируем и разблокируем
	err := storage.LocalRegister(user, password)
	require.NoError(t, err)
	err = storage.Unlock(user, password)
	require.NoError(t, err)

	// Получаем время синхронизации до сохранения
	retrievedTime, err := storage.GetLastSyncTime()
	require.NoError(t, err)
	assert.Equal(t, int64(0), retrievedTime)
}

func TestClose(t *testing.T) {
	storage, _ := setupTestDB(t)

	err := storage.Close()
	assert.NoError(t, err)

	// Повторное закрытие не должно вызывать ошибку
	err = storage.Close()
	assert.NoError(t, err)
}
