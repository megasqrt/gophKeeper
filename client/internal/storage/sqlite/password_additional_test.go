package sqlite

import (
	"testing"

	models "gophKeeper/pkg/grpchelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatePass_WithServerID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сохраняем пароль
	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		ServerID:    100,
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)

	localID := pass.LocalID

	// Обновляем пароль с ServerID
	pass.ServerID = 200
	pass.Login = "updatedlogin"
	pass.Password = "updatedpassword"
	pass.Description = "updated description"

	err = storage.UpdatePass(pass)
	require.NoError(t, err)
	assert.Equal(t, localID, pass.LocalID)
	assert.Equal(t, int64(200), pass.ServerID)

	// Проверяем обновление
	passwords, err := storage.GetPasswords()
	require.NoError(t, err)

	found := false
	for _, p := range passwords {
		if p.LocalID == localID {
			found = true
			assert.Equal(t, "updatedlogin", p.Login)
			assert.Equal(t, "updatedpassword", p.Password)
			assert.Equal(t, int64(200), p.ServerID)
			break
		}
	}
	assert.True(t, found, "Updated password not found")
}

func TestGetPasswords_WithServerID(t *testing.T) {
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

	// Получаем пароли
	passwords, err := storage.GetPasswords()
	require.NoError(t, err)

	found := false
	for _, p := range passwords {
		if p.Login == "testlogin" {
			found = true
			assert.Equal(t, int64(999), p.ServerID)
			break
		}
	}
	assert.True(t, found, "Password with ServerID not found")
}

func TestSavePass_WithoutServerID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		ServerID:    0, // Нет ServerID
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), pass.LocalID)
}

func TestSavePass_WithServerID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	pass := &models.Password{
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		ServerID:    12345,
	}

	err := storage.SavePass(pass)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), pass.LocalID)
	assert.Equal(t, int64(12345), pass.ServerID)
}

func TestUpdatePass_WithDeleted(t *testing.T) {
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

	// Обновляем пароль с пометкой удаления
	pass.Deleted = true
	err = storage.UpdatePass(pass)
	require.NoError(t, err)

	// Проверяем, что пароль не возвращается
	passwords, err := storage.GetPasswords()
	require.NoError(t, err)

	for _, p := range passwords {
		assert.NotEqual(t, localID, p.LocalID)
	}
}

func TestGetPasswords_MultiplePasswords(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сохраняем несколько паролей
	for i := 0; i < 5; i++ {
		pass := &models.Password{
			Login:       "login" + string(rune('0'+i)),
			Password:    "password" + string(rune('0'+i)),
			Description: "description" + string(rune('0'+i)),
			ServerID:    int64(i + 1),
		}
		err := storage.SavePass(pass)
		require.NoError(t, err)
	}

	// Получаем все пароли
	passwords, err := storage.GetPasswords()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(passwords), 5)
}

func TestDeletePass_NonExistent(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Пытаемся удалить несуществующий пароль
	err := storage.DeletePass(99999)
	// Не должно быть ошибки, просто ничего не удалится
	assert.NoError(t, err)
}

func TestFindPasswordByServerID_MultiplePasswords(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Сохраняем несколько паролей с разными ServerID
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

	// Ищем по ServerID
	foundLocalID1, err := storage.FindPasswordByServerID(100)
	require.NoError(t, err)
	assert.Equal(t, pass1.LocalID, foundLocalID1)

	foundLocalID2, err := storage.FindPasswordByServerID(200)
	require.NoError(t, err)
	assert.Equal(t, pass2.LocalID, foundLocalID2)
}
