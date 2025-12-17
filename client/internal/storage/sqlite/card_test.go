package sqlite

import (
	"testing"

	models "gophKeeper/pkg/grpchelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveCard_GetCards(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	card := &models.Card{
		Number:   "1234567890123456",
		Holder:   "Test Holder",
		Expiry:   "12/25",
		CVV:      "123",
		Metadata: "test metadata",
	}

	err := storage.SaveCard(card)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), card.LocalID)
	assert.NotEmpty(t, card.Checksum)

	// Получаем карты
	cards, err := storage.GetCards()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(cards), 1)

	found := false
	for _, c := range cards {
		if c.Number == "1234567890123456" {
			found = true
			assert.Equal(t, "Test Holder", c.Holder)
			assert.Equal(t, "12/25", c.Expiry)
			assert.Equal(t, "123", c.CVV)
			break
		}
	}
	assert.True(t, found, "Card not found")
}

func TestUpdateCard(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	card := &models.Card{
		Number: "1234567890123456",
		Holder: "Original Holder",
		Expiry: "12/25",
		CVV:    "123",
	}

	err := storage.SaveCard(card)
	require.NoError(t, err)

	localID := card.LocalID

	// Обновляем карту
	card.Holder = "Updated Holder"
	err = storage.UpdateCard(card)
	require.NoError(t, err)
	assert.Equal(t, localID, card.LocalID)

	// Проверяем обновление
	cards, err := storage.GetCards()
	require.NoError(t, err)

	found := false
	for _, c := range cards {
		if c.LocalID == localID {
			found = true
			assert.Equal(t, "Updated Holder", c.Holder)
			break
		}
	}
	assert.True(t, found)
}

func TestDeleteCard(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	card := &models.Card{
		Number: "1234567890123456",
		Holder: "Test Holder",
		Expiry: "12/25",
		CVV:    "123",
	}

	err := storage.SaveCard(card)
	require.NoError(t, err)

	localID := card.LocalID

	// Удаляем карту
	err = storage.DeleteCard(localID)
	require.NoError(t, err)

	// Проверяем, что карта не возвращается
	cards, err := storage.GetCards()
	require.NoError(t, err)

	for _, c := range cards {
		assert.NotEqual(t, localID, c.LocalID)
	}
}

func TestSaveText_GetTexts(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	text := &models.TextData{
		Title: "Test Title",
		Text:  "Test content",
	}

	err := storage.SaveText(text)
	require.NoError(t, err)
	assert.NotEqual(t, int64(0), text.LocalID)
	assert.NotEmpty(t, text.Checksum)

	// Получаем тексты
	texts, err := storage.GetTexts()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(texts), 1)

	found := false
	for _, textItem := range texts {
		if textItem.Title == "Test Title" {
			found = true
			assert.Equal(t, "Test content", textItem.Text)
			break
		}
	}
	assert.True(t, found, "Text not found")
}

func TestUpdateText(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	text := &models.TextData{
		Title: "Original Title",
		Text:  "Original content",
	}

	err := storage.SaveText(text)
	require.NoError(t, err)

	localID := text.LocalID

	// Обновляем текст
	text.Title = "Updated Title"
	text.Text = "Updated content"
	err = storage.UpdateText(text)
	require.NoError(t, err)
	assert.Equal(t, localID, text.LocalID)

	// Проверяем обновление
	texts, err := storage.GetTexts()
	require.NoError(t, err)

	found := false
	for _, textItem := range texts {
		if textItem.LocalID == localID {
			found = true
			assert.Equal(t, "Updated Title", textItem.Title)
			assert.Equal(t, "Updated content", textItem.Text)
			break
		}
	}
	assert.True(t, found)
}

func TestDeleteText(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	text := &models.TextData{
		Title: "Test Title",
		Text:  "Test content",
	}

	err := storage.SaveText(text)
	require.NoError(t, err)

	localID := text.LocalID

	// Удаляем текст
	err = storage.DeleteText(localID)
	require.NoError(t, err)

	// Проверяем, что текст не возвращается
	texts, err := storage.GetTexts()
	require.NoError(t, err)

	for _, textItem := range texts {
		assert.NotEqual(t, localID, textItem.LocalID)
	}
}

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
