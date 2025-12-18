package sqlite

import (
	"testing"

	models "gophKeeper/pkg/grpchelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveFileMetadata_GetFiles(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	file := &models.FileData{
		Name:     "test.txt",
		Size:     1024,
		Metadata: "test metadata",
	}

	err := storage.SaveFileMetadata(file)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), file.LocalID)
	assert.NotEmpty(t, file.Checksum)

	// Получаем файлы
	files, err := storage.GetFiles()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)

	found := false
	for _, f := range files {
		if f.Name == "test.txt" {
			found = true
			assert.Equal(t, int64(1024), f.Size)
			break
		}
	}
	assert.True(t, found, "File not found")
}

func TestSaveFile(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	file := &models.FileData{
		Name:     "test.txt",
		Size:     1024,
		Metadata: "test metadata",
	}
	content := []byte("test file content")

	err := storage.SaveFile(file, content)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), file.LocalID)
}

func TestUpdateFile(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	file := &models.FileData{
		Name:     "test.txt",
		Size:     1024,
		Metadata: "original metadata",
	}

	err := storage.SaveFileMetadata(file)
	require.NoError(t, err)

	localID := file.LocalID

	// Обновляем файл
	file.Metadata = "updated metadata"
	file.Size = 2048
	err = storage.UpdateFile(file)
	require.NoError(t, err)
	assert.Equal(t, localID, file.LocalID)
}

func TestDeleteFileByID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	file := &models.FileData{
		Name: "test.txt",
		Size: 1024,
	}

	err := storage.SaveFileMetadata(file)
	require.NoError(t, err)

	localID := file.LocalID

	// Удаляем файл
	err = storage.DeleteFileByID(localID)
	require.NoError(t, err)

	// Проверяем, что файл не возвращается
	files, err := storage.GetFiles()
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, localID, f.LocalID)
	}
}

func TestGetFilesByIDs(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем несколько файлов
	file1 := &models.FileData{
		Name:     "file1.txt",
		Size:     1024,
		Metadata: "metadata1",
	}
	file2 := &models.FileData{
		Name:     "file2.txt",
		Size:     2048,
		Metadata: "metadata2",
	}
	file3 := &models.FileData{
		Name:     "file3.txt",
		Size:     3072,
		Metadata: "metadata3",
	}

	err := storage.SaveFileMetadata(file1)
	require.NoError(t, err)
	err = storage.SaveFileMetadata(file2)
	require.NoError(t, err)
	err = storage.SaveFileMetadata(file3)
	require.NoError(t, err)

	// Получаем файлы по IDs (метод пока не реализован, возвращает nil)
	ids := []int64{file1.LocalID, file3.LocalID}
	files, err := storage.GetFilesByIDs(ids)
	require.NoError(t, err)
	// Метод пока не реализован, возвращает nil
	assert.Nil(t, files)
}

func TestGetFileByID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем файл
	file := &models.FileData{
		Name:     "test.txt",
		Size:     1024,
		Metadata: "test metadata",
	}
	content := []byte("test file content")

	err := storage.SaveFile(file, content)
	require.NoError(t, err)

	// Получаем файл по ID (метод пока не реализован, возвращает nil)
	result, err := storage.GetFileByID(file.LocalID)
	require.NoError(t, err)
	// Метод пока не реализован, возвращает nil
	assert.Nil(t, result)
}

func TestGetFiles_WithDeleted(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем файл
	file := &models.FileData{
		Name:     "test.txt",
		Size:     1024,
		Metadata: "test metadata",
	}

	err := storage.SaveFileMetadata(file)
	require.NoError(t, err)

	localID := file.LocalID

	// Удаляем файл
	err = storage.DeleteFileByID(localID)
	require.NoError(t, err)

	// Проверяем, что удаленный файл не возвращается
	files, err := storage.GetFiles()
	require.NoError(t, err)

	for _, f := range files {
		assert.NotEqual(t, localID, f.LocalID, "Deleted file should not be returned")
	}
}

func TestGetFiles_WithServerID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем файл с ServerID
	file := &models.FileData{
		Name:     "test.txt",
		Size:     1024,
		Metadata: "test metadata",
		ServerID: 999,
	}

	err := storage.SaveFileMetadata(file)
	require.NoError(t, err)

	// Получаем файлы
	files, err := storage.GetFiles()
	require.NoError(t, err)

	found := false
	for _, f := range files {
		if f.LocalID == file.LocalID {
			found = true
			assert.Equal(t, int64(999), f.ServerID)
			break
		}
	}
	assert.True(t, found, "File with ServerID not found")
}

func TestSaveFileMetadata_UpdateExisting(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем файл
	file := &models.FileData{
		Name:     "original.txt",
		Size:     1024,
		Metadata: "original metadata",
	}

	err := storage.SaveFileMetadata(file)
	require.NoError(t, err)

	localID := file.LocalID

	// Обновляем файл
	file.Name = "updated.txt"
	file.Size = 2048
	file.Metadata = "updated metadata"
	file.ServerID = 888

	err = storage.SaveFileMetadata(file)
	require.NoError(t, err)
	assert.Equal(t, localID, file.LocalID)

	// Проверяем обновление
	files, err := storage.GetFiles()
	require.NoError(t, err)

	found := false
	for _, f := range files {
		if f.LocalID == localID {
			found = true
			assert.Equal(t, "updated.txt", f.Name)
			assert.Equal(t, int64(2048), f.Size)
			assert.Equal(t, "updated metadata", f.Metadata)
			assert.Equal(t, int64(888), f.ServerID)
			break
		}
	}
	assert.True(t, found, "Updated file not found")
}

func TestSaveFile_WithServerID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	file := &models.FileData{
		Name:     "test.txt",
		Size:     1024,
		Metadata: "test metadata",
		ServerID: 777,
	}
	content := []byte("test file content")

	err := storage.SaveFile(file, content)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), file.LocalID)
	assert.Equal(t, int64(777), file.ServerID)
}

func TestSaveFile_WithEmptyContent(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	file := &models.FileData{
		Name:     "empty.txt",
		Size:     0,
		Metadata: "empty file",
	}
	content := []byte{}

	err := storage.SaveFile(file, content)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), file.LocalID)
}

func TestGetFiles_MultipleFiles(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем несколько файлов
	for i := 0; i < 5; i++ {
		file := &models.FileData{
			Name:     "file" + string(rune('0'+i)) + ".txt",
			Size:     int64(1024 * (i + 1)),
			Metadata: "metadata" + string(rune('0'+i)),
		}
		err := storage.SaveFileMetadata(file)
		require.NoError(t, err)
	}

	// Получаем все файлы
	files, err := storage.GetFiles()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 5)
}

func TestSaveFileMetadata_WithEmptyMetadata(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	file := &models.FileData{
		Name:     "test.txt",
		Size:     1024,
		Metadata: "", // Пустые метаданные
	}

	err := storage.SaveFileMetadata(file)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), file.LocalID)

	// Проверяем, что файл сохранился
	files, err := storage.GetFiles()
	require.NoError(t, err)

	found := false
	for _, f := range files {
		if f.LocalID == file.LocalID {
			found = true
			assert.Equal(t, "", f.Metadata)
			break
		}
	}
	assert.True(t, found, "File with empty metadata not found")
}
