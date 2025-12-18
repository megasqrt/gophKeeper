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

type DeviceRepoTestSuite struct {
	postgresContainer testcontainers.Container
	db                *sqlx.DB
	suite.Suite
}

func (suite *DeviceRepoTestSuite) SetupSuite() {
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

func (suite *DeviceRepoTestSuite) TearDownSuite() {
	ctx := context.Background()
	suite.db.Close()
	suite.Require().NoError(suite.postgresContainer.Terminate(ctx))
}

func (suite *DeviceRepoTestSuite) SetupTest() {
	// Очищаем таблицы перед каждым тестом
	suite.db.Exec("DELETE FROM devices")
	suite.db.Exec("DELETE FROM users")
}

func (suite *DeviceRepoTestSuite) TestCreate() {
	ctx := context.Background()
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	device := &model.Device{
		ID:         uuid.New(),
		UserID:     userID,
		DeviceName: "Test Device",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	err = deviceRepo.Create(ctx, device)
	suite.Require().NoError(err)
}

func (suite *DeviceRepoTestSuite) TestUpdate() {
	ctx := context.Background()
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	device := &model.Device{
		ID:         uuid.New(),
		UserID:     userID,
		DeviceName: "Original Device",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	err = deviceRepo.Create(ctx, device)
	suite.Require().NoError(err)

	// Обновляем устройство
	device.DeviceName = "Updated Device"
	device.UpdatedAt = time.Now().Unix()

	err = deviceRepo.Update(ctx, device)
	suite.Require().NoError(err)

	// Проверяем обновление
	found, err := deviceRepo.FindByID(ctx, device.ID)
	suite.Require().NoError(err)
	suite.Assert().Equal("Updated Device", found.DeviceName)
}

func (suite *DeviceRepoTestSuite) TestFindByID() {
	ctx := context.Background()
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	deviceID := uuid.New()
	device := &model.Device{
		ID:         deviceID,
		UserID:     userID,
		DeviceName: "Test Device",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	err = deviceRepo.Create(ctx, device)
	suite.Require().NoError(err)

	// Ищем устройство
	found, err := deviceRepo.FindByID(ctx, deviceID)
	suite.Require().NoError(err)
	suite.Require().NotNil(found)
	suite.Assert().Equal(deviceID, found.ID)
	suite.Assert().Equal("Test Device", found.DeviceName)
}

func (suite *DeviceRepoTestSuite) TestFindByID_NotFound() {
	ctx := context.Background()
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	nonExistentID := uuid.New()
	_, err := deviceRepo.FindByID(ctx, nonExistentID)
	suite.Require().Error(err)
	suite.Assert().Contains(err.Error(), "device not found")
}

func (suite *DeviceRepoTestSuite) TestFindByUserID() {
	ctx := context.Background()
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	// Создаем несколько устройств
	device1 := &model.Device{
		ID:         uuid.New(),
		UserID:     userID,
		DeviceName: "Device 1",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	device2 := &model.Device{
		ID:         uuid.New(),
		UserID:     userID,
		DeviceName: "Device 2",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	err = deviceRepo.Create(ctx, device1)
	suite.Require().NoError(err)

	err = deviceRepo.Create(ctx, device2)
	suite.Require().NoError(err)

	// Ищем все устройства пользователя
	devices, err := deviceRepo.FindByUserID(ctx, userID)
	suite.Require().NoError(err)
	suite.Assert().GreaterOrEqual(len(devices), 2)

	found1 := false
	found2 := false
	for _, d := range devices {
		if d.ID == device1.ID {
			found1 = true
		}
		if d.ID == device2.ID {
			found2 = true
		}
	}
	suite.Assert().True(found1, "Device 1 not found")
	suite.Assert().True(found2, "Device 2 not found")
}

func (suite *DeviceRepoTestSuite) TestDelete() {
	ctx := context.Background()
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	deviceID := uuid.New()
	device := &model.Device{
		ID:         deviceID,
		UserID:     userID,
		DeviceName: "Test Device",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	err = deviceRepo.Create(ctx, device)
	suite.Require().NoError(err)

	// Удаляем устройство
	err = deviceRepo.Delete(ctx, deviceID)
	suite.Require().NoError(err)

	// Проверяем, что устройство удалено
	_, err = deviceRepo.FindByID(ctx, deviceID)
	suite.Require().Error(err)
	suite.Assert().Contains(err.Error(), "device not found")
}

func (suite *DeviceRepoTestSuite) TestSyncTime() {
	ctx := context.Background()
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	deviceID := uuid.New()
	device := &model.Device{
		ID:         deviceID,
		UserID:     userID,
		DeviceName: "Test Device",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	err = deviceRepo.Create(ctx, device)
	suite.Require().NoError(err)

	// Синхронизируем время
	err = deviceRepo.SyncTime(ctx, deviceID)
	suite.Require().NoError(err)

	// Проверяем, что last_sync установлен (через FindByID)
	found, err := deviceRepo.FindByID(ctx, deviceID)
	suite.Require().NoError(err)
	suite.Assert().NotNil(found)
	suite.Assert().NotNil(found.LastSync)
	suite.Assert().Greater(*found.LastSync, int64(0))
}

func (suite *DeviceRepoTestSuite) TestUpdate_NotFound() {
	ctx := context.Background()
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	nonExistentDevice := &model.Device{
		ID:         uuid.New(),
		DeviceName: "Non-existent",
		UpdatedAt:  time.Now().Unix(),
	}

	err := deviceRepo.Update(ctx, nonExistentDevice)
	suite.Require().Error(err)
	suite.Assert().Contains(err.Error(), "no rows were updated")
}

func TestDeviceRepoTestSuite(t *testing.T) {
	suite.Run(t, new(DeviceRepoTestSuite))
}
