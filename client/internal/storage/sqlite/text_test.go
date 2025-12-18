package sqlite

import (
	"testing"

	models "gophKeeper/pkg/grpchelper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveText_GetTexts(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	text := &models.TextData{
		Title: "Test Title",
		Data:  "Test content",
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
			assert.Equal(t, "Test content", textItem.Data)
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
		Data:  "Original content",
	}

	err := storage.SaveText(text)
	require.NoError(t, err)

	localID := text.LocalID

	// Обновляем текст
	text.Title = "Updated Title"
	text.Data = "Updated content"
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
			assert.Equal(t, "Updated content", textItem.Data)
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
		Data:  "Test content",
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

func TestGetTextsByIDs(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем несколько текстов
	text1 := &models.TextData{
		Title: "Title 1",
		Data:  "Content 1",
	}
	text2 := &models.TextData{
		Title: "Title 2",
		Data:  "Content 2",
	}
	text3 := &models.TextData{
		Title: "Title 3",
		Data:  "Content 3",
	}

	err := storage.SaveText(text1)
	require.NoError(t, err)
	err = storage.SaveText(text2)
	require.NoError(t, err)
	err = storage.SaveText(text3)
	require.NoError(t, err)

	// Получаем тексты по IDs (метод пока не реализован, возвращает nil)
	ids := []int64{text1.LocalID, text3.LocalID}
	texts, err := storage.GetTextsByIDs(ids)
	require.NoError(t, err)
	// Метод пока не реализован, возвращает nil
	assert.Nil(t, texts)
}
