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

func TestGetCardsByIDs(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем несколько карт
	card1 := &models.Card{
		Number:   "1111111111111111",
		Holder:   "Holder 1",
		Expiry:   "01/25",
		CVV:      "111",
		Metadata: "metadata1",
	}
	card2 := &models.Card{
		Number:   "2222222222222222",
		Holder:   "Holder 2",
		Expiry:   "02/25",
		CVV:      "222",
		Metadata: "metadata2",
	}
	card3 := &models.Card{
		Number:   "3333333333333333",
		Holder:   "Holder 3",
		Expiry:   "03/25",
		CVV:      "333",
		Metadata: "metadata3",
	}

	err := storage.SaveCard(card1)
	require.NoError(t, err)
	err = storage.SaveCard(card2)
	require.NoError(t, err)
	err = storage.SaveCard(card3)
	require.NoError(t, err)

	// Получаем карты по IDs (метод пока не реализован, возвращает nil)
	ids := []int64{card1.LocalID, card3.LocalID}
	cards, err := storage.GetCardsByIDs(ids)
	require.NoError(t, err)
	// Метод пока не реализован, возвращает nil
	assert.Nil(t, cards)
}

func TestFindCardByServerID(t *testing.T) {
	storage := setupTestStorageWithUnlock(t)
	defer storage.Close()

	// Создаем карту с ServerID
	card := &models.Card{
		Number:   "1234567890123456",
		Holder:   "Test Holder",
		Expiry:   "12/25",
		CVV:      "123",
		ServerID: 999,
	}

	err := storage.SaveCard(card)
	require.NoError(t, err)

	// Ищем карту по ServerID
	foundLocalID, err := storage.FindCardByServerID(999)
	require.NoError(t, err)
	assert.Equal(t, card.LocalID, foundLocalID)

	// Ищем несуществующий ServerID
	foundLocalID, err = storage.FindCardByServerID(888)
	require.NoError(t, err)
	assert.Equal(t, int64(0), foundLocalID)

	// Удаляем карту и проверяем, что она не находится
	err = storage.DeleteCard(card.LocalID)
	require.NoError(t, err)

	foundLocalID, err = storage.FindCardByServerID(999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), foundLocalID, "Deleted card should not be found")
}
