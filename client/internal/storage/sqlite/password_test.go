package sqlite

import (
	"testing"
	"time"

	models "gophKeeper/pkg/grpchelper"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorageWithUnlock(t *testing.T) *SqliteStorage {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	log := zerolog.Nop()

	storage, err := NewSqliteStorage(dbPath, log)
	require.NoError(t, err)

	// Создаем таблицы
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
			client_version INTEGER
		);
		
		CREATE TABLE IF NOT EXISTS cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id INTEGER,
			card_number_data BLOB NOT NULL,
			card_holder_data BLOB NOT NULL,
			expiry_date_data BLOB NOT NULL,
			cvc_data BLOB NOT NULL,
			metadata BLOB,
			checksum TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER
		);
		
		CREATE TABLE IF NOT EXISTS note (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id INTEGER,
			title BLOB NOT NULL,
			text BLOB NOT NULL,
			checksum TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER
		);
		
		CREATE TABLE IF NOT EXISTS binary_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id INTEGER,
			data BLOB NOT NULL,
			name BLOB NOT NULL,
			metadata BLOB,
			size INTEGER,
			checksum TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER
		);
	`)
	require.NoError(t, err)

	user := "testuser"
	password := "testpassword"

	err = storage.LocalRegister(user, password)
	require.NoError(t, err)

	err = storage.Unlock(user, password)
	require.NoError(t, err)

	return storage
}

func TestSavePass(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		ServerID:    123,
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), pass.LocalID)
	assert.NotEmpty(t, pass.Checksum)
	assert.NotEqual(t, int64(0), pass.ChangeTime)
}

func TestSavePass_WithDeleted(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		Deleted:     true,
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)
}

func TestUpdatePass_Success(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сначала сохраняем пароль
	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)

	localID := pass.LocalID

	// Обновляем пароль
	pass.Login = "updatedlogin"
	pass.Password = "updatedpassword"
	pass.Description = "updated description"
	pass.ServerID = 456

	err = storage.UpdatePass(pass)
	require.NoError(t, err)
	assert.Equal(t, localID, pass.LocalID)
}

func TestUpdatePass_NoLocalID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		LocalID:     0, // Нет LocalID
	}

	err := storage.UpdatePass(pass)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LocalID is required")
}

func TestGetPasswords(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сохраняем несколько паролей
	pass1 := &models.Password{
		Login:       "login1",
		Password:    "password1",
		Description: "description1",
		ServerID:    100,
	}

	pass2 := &models.Password{
		Login:       "login2",
		Password:    "password2",
		Description: "description2",
		ServerID:    200,
	}

	err := storage.SavePass(pass1)
	require.NoError(t, err)

	err = storage.SavePass(pass2)
	require.NoError(t, err)

	// Получаем все пароли
	passwords, err := storage.GetPasswords()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(passwords), 2)

	// Проверяем, что пароли корректно расшифрованы
	found1 := false
	found2 := false
	for _, p := range passwords {
		if p.Login == "login1" && p.Password == "password1" {
			found1 = true
			assert.Equal(t, int64(100), p.ServerID)
		}
		if p.Login == "login2" && p.Password == "password2" {
			found2 = true
			assert.Equal(t, int64(200), p.ServerID)
		}
	}
	assert.True(t, found1, "Password 1 not found")
	assert.True(t, found2, "Password 2 not found")
}

func TestGetPasswords_Empty(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	passwords, err := storage.GetPasswords()
	require.NoError(t, err)
	assert.Empty(t, passwords)
}

func TestDeletePass(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сохраняем пароль
	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)

	localID := pass.LocalID

	// Удаляем пароль
	err = storage.DeletePass(localID)
	require.NoError(t, err)

	// Проверяем, что пароль не возвращается в GetPasswords
	passwords, err := storage.GetPasswords()
	require.NoError(t, err)

	for _, p := range passwords {
		assert.NotEqual(t, localID, p.LocalID)
	}
}

func TestFindPasswordByServerID_Success(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сохраняем пароль с ServerID
	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		ServerID:    999,
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)

	localID := pass.LocalID

	// Ищем по ServerID
	foundLocalID, err := storage.FindPasswordByServerID(999)
	require.NoError(t, err)
	assert.Equal(t, localID, foundLocalID)
}

func TestFindPasswordByServerID_NotFound(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Ищем несуществующий ServerID
	foundLocalID, err := storage.FindPasswordByServerID(99999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), foundLocalID)
}

func TestFindPasswordByServerID_Deleted(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сохраняем пароль
	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		ServerID:    888,
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)

	localID := pass.LocalID

	// Удаляем пароль
	err = storage.DeletePass(localID)
	require.NoError(t, err)

	// Ищем удаленный пароль - FindPasswordByServerID ищет только не удаленные (deleted_at IS NULL)
	// поэтому удаленный пароль не должен находиться
	foundLocalID, err := storage.FindPasswordByServerID(888)
	require.NoError(t, err)
	// Должен вернуть 0, так как пароль удален и не попадает в условие deleted_at IS NULL
	assert.Equal(t, int64(0), foundLocalID, "Deleted password should not be found")
}

func TestCalculatePasswordChecksum(t *testing.T) {
	checksum1 := calculatePasswordChecksum("login1", "password1", "desc1")
	checksum2 := calculatePasswordChecksum("login2", "password2", "desc2")
	checksum3 := calculatePasswordChecksum("login1", "password1", "desc1")

	assert.NotEmpty(t, checksum1)
	assert.NotEmpty(t, checksum2)
	assert.NotEqual(t, checksum1, checksum2)
	assert.Equal(t, checksum1, checksum3) // Одинаковые данные = одинаковый checksum
}

func TestEncryptPasswordData_DecryptPasswordData(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
	}

	// Шифруем
	encrypted, err := storage.encryptPasswordData(pass)
	require.NoError(t, err)
	assert.NotNil(t, encrypted)

	// Расшифровываем
	decrypted, err := storage.decryptPasswordData(encrypted)
	require.NoError(t, err)
	assert.Equal(t, pass.Login, decrypted.Login)
	assert.Equal(t, pass.Password, decrypted.Password)
	assert.Equal(t, pass.Description, decrypted.Description)
}

func TestSavePass_UpdatePass_GetPasswords_Integration(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сохраняем пароль
	pass := &models.Password{
		Login:       "originallogin",
		Password:    "originalpassword",
		Description: "original description",
		ServerID:    111,
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)

	originalLocalID := pass.LocalID
	originalChecksum := pass.Checksum

	// Обновляем пароль
	pass.Login = "updatedlogin"
	pass.Password = "updatedpassword"
	pass.Description = "updated description"
	pass.ServerID = 222

	err = storage.UpdatePass(pass)
	require.NoError(t, err)

	// Проверяем, что LocalID не изменился, но checksum изменился
	assert.Equal(t, originalLocalID, pass.LocalID)
	assert.NotEqual(t, originalChecksum, pass.Checksum)

	// Получаем пароли и проверяем обновление
	passwords, err := storage.GetPasswords()
	require.NoError(t, err)

	found := false
	for _, p := range passwords {
		if p.LocalID == originalLocalID {
			found = true
			assert.Equal(t, "updatedlogin", p.Login)
			assert.Equal(t, "updatedpassword", p.Password)
			assert.Equal(t, "updated description", p.Description)
			assert.Equal(t, int64(222), p.ServerID)
			break
		}
	}
	assert.True(t, found, "Updated password not found")
}

func TestSavePass_WithCreateTime(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	createTime := time.Now().Unix() - 1000 // 1000 секунд назад

	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		CreateTime:  createTime,
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)

	// Проверяем, что CreateTime сохранился
	passwords, err := storage.GetPasswords()
	require.NoError(t, err)

	found := false
	for _, p := range passwords {
		if p.Login == "testlogin" {
			found = true
			assert.Equal(t, createTime, p.CreateTime)
			break
		}
	}
	assert.True(t, found)
}
