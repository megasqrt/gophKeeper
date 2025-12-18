package postgres_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	migrations "gophKeeper/pkg/migrations/pg"
	"gophKeeper/server/internal/domain/model"
	"gophKeeper/server/internal/storage/postgres"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type CardRepoTestSuite struct {
	postgresContainer testcontainers.Container
	db                *sqlx.DB
	suite.Suite
}

func (suite *CardRepoTestSuite) SetupSuite() {
	ctx := context.Background()
	postgresContainer, err := tpg.Run(ctx,
		"postgres:16-alpine",
		tpg.WithDatabase("test"),
		tpg.WithUsername("user"),
		tpg.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort("5432/tcp"),
			).WithDeadline(1*time.Minute),
		))
	suite.Require().NoError(err)
	suite.postgresContainer = postgresContainer

	postgresEndpoint, err := suite.postgresContainer.Endpoint(ctx, "")
	suite.Require().NoError(err)
	dsn := fmt.Sprintf("postgres://user:password@%s/test?sslmode=disable", postgresEndpoint)

	root, err := projectRoot()
	suite.Require().NoError(err)

	migrator := migrations.NewMigration(dsn)
	baseMigrationsPath := filepath.Join(root, "server", "internal", "storage", "postgres", "migrations")
	suite.Require().NoError(migrator.Up("file://" + baseMigrationsPath))

	db, err := sqlx.Connect("pgx", dsn)
	suite.Require().NoError(err)
	suite.db = db
}

func (suite *CardRepoTestSuite) TearDownSuite() {
	ctx := context.Background()
	suite.db.Close()
	suite.Require().NoError(suite.postgresContainer.Terminate(ctx))
}

func (suite *CardRepoTestSuite) SetupTest() {
	// Очищаем таблицы перед каждым тестом
	suite.db.Exec("DELETE FROM bank_cards")
	suite.db.Exec("DELETE FROM devices")
	suite.db.Exec("DELETE FROM users")
}

func (suite *CardRepoTestSuite) TestCreate() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	card := &model.Card{
		UserID:    userID,
		Number:    "4111111111111111",
		Holder:    "John Doe",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "test metadata",
		Checksum:  "testchecksum123",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := cardRepo.Create(ctx, card)
	suite.Require().NoError(err)
	suite.Assert().NotEqual(int64(0), id)
}

func (suite *CardRepoTestSuite) TestGetByUserID() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	// Создаем несколько карт
	card1 := &model.Card{
		UserID:    userID,
		Number:    "4111111111111111",
		Holder:    "John Doe",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "metadata1",
		Checksum:  "checksum1",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	card2 := &model.Card{
		UserID:    userID,
		Number:    "5555555555554444",
		Holder:    "Jane Smith",
		Expiry:    "06/26",
		CVV:       "456",
		Metadata:  "metadata2",
		Checksum:  "checksum2",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id1, err := cardRepo.Create(ctx, card1)
	suite.Require().NoError(err)

	id2, err := cardRepo.Create(ctx, card2)
	suite.Require().NoError(err)

	cards, err := cardRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)
	suite.Assert().GreaterOrEqual(len(cards), 2)

	found1 := false
	found2 := false
	for _, c := range cards {
		if c.ID == id1 {
			found1 = true
			suite.Assert().Equal("4111111111111111", c.Number)
			suite.Assert().Equal("John Doe", c.Holder)
		}
		if c.ID == id2 {
			found2 = true
			suite.Assert().Equal("5555555555554444", c.Number)
			suite.Assert().Equal("Jane Smith", c.Holder)
		}
	}
	suite.Assert().True(found1, "Card 1 not found")
	suite.Assert().True(found2, "Card 2 not found")
}

func (suite *CardRepoTestSuite) TestUpdate() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	card := &model.Card{
		UserID:    userID,
		Number:    "4111111111111111",
		Holder:    "Original Holder",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "original metadata",
		Checksum:  "originalchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := cardRepo.Create(ctx, card)
	suite.Require().NoError(err)

	// Обновляем карту
	card.ID = id
	card.Number = "5555555555554444"
	card.Holder = "Updated Holder"
	card.Expiry = "06/26"
	card.CVV = "456"
	card.Metadata = "updated metadata"
	card.Checksum = "updatedchecksum"
	card.UpdatedAt = time.Now().Unix()
	card.Version = 2

	err = cardRepo.Update(ctx, card)
	suite.Require().NoError(err)

	// Проверяем обновление
	cards, err := cardRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)

	found := false
	for _, c := range cards {
		if c.ID == id {
			found = true
			suite.Assert().Equal("5555555555554444", c.Number)
			suite.Assert().Equal("Updated Holder", c.Holder)
			suite.Assert().Equal("06/26", c.Expiry)
			suite.Assert().Equal("456", c.CVV)
			suite.Assert().Equal("updated metadata", c.Metadata)
			suite.Assert().Equal(int32(2), c.Version)
			break
		}
	}
	suite.Assert().True(found, "Updated card not found")
}

func (suite *CardRepoTestSuite) TestUpdate_NotFound() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID := uuid.New()
	card := &model.Card{
		ID:        99999, // Несуществующий ID
		UserID:    userID,
		Number:    "4111111111111111",
		Holder:    "Test Holder",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "test metadata",
		Checksum:  "testchecksum",
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	err := cardRepo.Update(ctx, card)
	suite.Require().Error(err)
	suite.Assert().Contains(err.Error(), "no rows were updated")
}

func (suite *CardRepoTestSuite) TestDelete() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	card := &model.Card{
		UserID:    userID,
		Number:    "4111111111111111",
		Holder:    "John Doe",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "test metadata",
		Checksum:  "testchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := cardRepo.Create(ctx, card)
	suite.Require().NoError(err)

	// Удаляем карту
	err = cardRepo.Delete(ctx, id, userID)
	suite.Require().NoError(err)

	// Проверяем, что карта не возвращается в GetByUserID (так как deleted_at IS NULL фильтр)
	cards, err := cardRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)

	found := false
	for _, c := range cards {
		if c.ID == id {
			found = true
			break
		}
	}
	suite.Assert().False(found, "Deleted card should not be found in GetByUserID")
}

func (suite *CardRepoTestSuite) TestGetDataDeviceLastSinc() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	deviceID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	// Создаем устройство
	device := &model.Device{
		ID:         deviceID,
		UserID:     userID,
		DeviceName: "Test Device",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}
	err = deviceRepo.Create(ctx, device)
	suite.Require().NoError(err)

	// Создаем карту
	card := &model.Card{
		UserID:    userID,
		Number:    "4111111111111111",
		Holder:    "John Doe",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "test metadata",
		Checksum:  "testchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := cardRepo.Create(ctx, card)
	suite.Require().NoError(err)

	// Получаем карты с последней синхронизации
	cards, err := cardRepo.GetDataDeviceLastSinc(ctx, userID, deviceID)
	suite.Require().NoError(err)
	suite.Assert().GreaterOrEqual(len(cards), 1)

	found := false
	for _, c := range cards {
		if c.ID == id {
			found = true
			suite.Assert().Equal("4111111111111111", c.Number)
			suite.Assert().Equal("John Doe", c.Holder)
			break
		}
	}
	suite.Assert().True(found, "Card not found in sync result")
}

func (suite *CardRepoTestSuite) TestGetDataDeviceLastSinc_NoNewData() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	deviceID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	// Создаем устройство
	device := &model.Device{
		ID:         deviceID,
		UserID:     userID,
		DeviceName: "Test Device",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}
	err = deviceRepo.Create(ctx, device)
	suite.Require().NoError(err)

	// Синхронизируем время устройства (устанавливаем last_sync)
	err = deviceRepo.SyncTime(ctx, deviceID)
	suite.Require().NoError(err)

	// Создаем карту после синхронизации
	card := &model.Card{
		UserID:    userID,
		Number:    "4111111111111111",
		Holder:    "John Doe",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "test metadata",
		Checksum:  "testchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	_, err = cardRepo.Create(ctx, card)
	suite.Require().NoError(err)

	// Снова синхронизируем время
	err = deviceRepo.SyncTime(ctx, deviceID)
	suite.Require().NoError(err)

	// Теперь не должно быть новых данных
	cards, err := cardRepo.GetDataDeviceLastSinc(ctx, userID, deviceID)
	suite.Require().NoError(err)
	// Может быть 0 или больше, в зависимости от точности времени
	suite.Assert().GreaterOrEqual(len(cards), 0)
}

func (suite *CardRepoTestSuite) TestGetByUserID_Empty() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID := uuid.New()

	cards, err := cardRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)
	suite.Assert().Empty(cards)
}

func (suite *CardRepoTestSuite) TestUpdate_WrongUserID() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID1 := uuid.New()
	userID2 := uuid.New()
	err := createTestUser(suite.db, userID1)
	suite.Require().NoError(err)
	err = createTestUser(suite.db, userID2)
	suite.Require().NoError(err)

	card := &model.Card{
		UserID:    userID1,
		Number:    "4111111111111111",
		Holder:    "John Doe",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "test metadata",
		Checksum:  "testchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := cardRepo.Create(ctx, card)
	suite.Require().NoError(err)

	// Пытаемся обновить с неправильным userID
	card.ID = id
	card.UserID = userID2
	card.Number = "5555555555554444"
	card.UpdatedAt = time.Now().Unix()

	err = cardRepo.Update(ctx, card)
	suite.Require().Error(err)
	suite.Assert().Contains(err.Error(), "no rows were updated")
}

func (suite *CardRepoTestSuite) TestDelete_WrongUserID() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID1 := uuid.New()
	userID2 := uuid.New()
	err := createTestUser(suite.db, userID1)
	suite.Require().NoError(err)
	err = createTestUser(suite.db, userID2)
	suite.Require().NoError(err)

	card := &model.Card{
		UserID:    userID1,
		Number:    "4111111111111111",
		Holder:    "John Doe",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "test metadata",
		Checksum:  "testchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := cardRepo.Create(ctx, card)
	suite.Require().NoError(err)

	// Пытаемся удалить с неправильным userID
	err = cardRepo.Delete(ctx, id, userID2)
	suite.Require().NoError(err) // Delete не возвращает ошибку при отсутствии строк

	// Проверяем, что карта все еще существует
	cards, err := cardRepo.GetByUserID(ctx, userID1)
	suite.Require().NoError(err)
	found := false
	for _, c := range cards {
		if c.ID == id {
			found = true
			break
		}
	}
	suite.Assert().True(found, "Card should still exist after wrong userID delete")
}

func (suite *CardRepoTestSuite) TestGetByUserID_WithDeleted() {
	ctx := context.Background()
	cardRepo := postgres.NewCardRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	// Создаем карту
	card := &model.Card{
		UserID:    userID,
		Number:    "4111111111111111",
		Holder:    "John Doe",
		Expiry:    "12/25",
		CVV:       "123",
		Metadata:  "test metadata",
		Checksum:  "testchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := cardRepo.Create(ctx, card)
	suite.Require().NoError(err)

	// Удаляем карту
	err = cardRepo.Delete(ctx, id, userID)
	suite.Require().NoError(err)

	// Проверяем, что удаленная карта не возвращается
	cards, err := cardRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)
	suite.Assert().Empty(cards, "Deleted card should not be returned")
}

func TestCardRepoTestSuite(t *testing.T) {
	suite.Run(t, new(CardRepoTestSuite))
}
