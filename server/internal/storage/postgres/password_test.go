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

type PasswordRepoTestSuite struct {
	postgresContainer testcontainers.Container
	db                *sqlx.DB
	suite.Suite
}

func (suite *PasswordRepoTestSuite) SetupSuite() {
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

func (suite *PasswordRepoTestSuite) TearDownSuite() {
	ctx := context.Background()
	suite.db.Close()
	suite.Require().NoError(suite.postgresContainer.Terminate(ctx))
}

func (suite *PasswordRepoTestSuite) SetupTest() {
	// Очищаем таблицы перед каждым тестом
	suite.db.Exec("DELETE FROM login_passwords")
	suite.db.Exec("DELETE FROM devices")
	suite.db.Exec("DELETE FROM users")
}

func (suite *PasswordRepoTestSuite) TestCreate() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)

	userID := uuid.New()
	pass := &model.Password{
		UserID:      userID,
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		Checksum:    "testchecksum123",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	id, err := passRepo.Create(ctx, pass)
	suite.Require().NoError(err)
	suite.Assert().NotEqual(int64(0), id)
}

func (suite *PasswordRepoTestSuite) TestGetByUserID() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)

	userID := uuid.New()

	// Создаем несколько паролей
	pass1 := &model.Password{
		UserID:      userID,
		Login:       "login1",
		Password:    "password1",
		Description: "description1",
		Checksum:    "checksum1",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	pass2 := &model.Password{
		UserID:      userID,
		Login:       "login2",
		Password:    "password2",
		Description: "description2",
		Checksum:    "checksum2",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	id1, err := passRepo.Create(ctx, pass1)
	suite.Require().NoError(err)

	id2, err := passRepo.Create(ctx, pass2)
	suite.Require().NoError(err)

	passwords, err := passRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)
	suite.Assert().GreaterOrEqual(len(passwords), 2)

	found1 := false
	found2 := false
	for _, p := range passwords {
		if p.ID == id1 {
			found1 = true
			suite.Assert().Equal("login1", p.Login)
			suite.Assert().Equal("password1", p.Password)
		}
		if p.ID == id2 {
			found2 = true
			suite.Assert().Equal("login2", p.Login)
			suite.Assert().Equal("password2", p.Password)
		}
	}
	suite.Assert().True(found1, "Password 1 not found")
	suite.Assert().True(found2, "Password 2 not found")
}

func (suite *PasswordRepoTestSuite) TestUpdate() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)

	userID := uuid.New()
	pass := &model.Password{
		UserID:      userID,
		Login:       "originallogin",
		Password:    "originalpassword",
		Description: "original description",
		Checksum:    "originalchecksum",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	id, err := passRepo.Create(ctx, pass)
	suite.Require().NoError(err)

	// Обновляем пароль
	pass.ID = id
	pass.Login = "updatedlogin"
	pass.Password = "updatedpassword"
	pass.Description = "updated description"
	pass.Checksum = "updatedchecksum"
	pass.UpdatedAt = time.Now().Unix()
	pass.Version = 2

	err = passRepo.Update(ctx, pass)
	suite.Require().NoError(err)

	// Проверяем обновление
	passwords, err := passRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)

	found := false
	for _, p := range passwords {
		if p.ID == id {
			found = true
			suite.Assert().Equal("updatedlogin", p.Login)
			suite.Assert().Equal("updatedpassword", p.Password)
			suite.Assert().Equal("updated description", p.Description)
			suite.Assert().Equal(int64(2), p.Version)
			break
		}
	}
	suite.Assert().True(found, "Updated password not found")
}

func (suite *PasswordRepoTestSuite) TestUpdate_NotFound() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)

	userID := uuid.New()
	pass := &model.Password{
		ID:          99999, // Несуществующий ID
		UserID:      userID,
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		Checksum:    "testchecksum",
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	err := passRepo.Update(ctx, pass)
	suite.Require().Error(err)
	suite.Assert().Contains(err.Error(), "no rows were updated")
}

func (suite *PasswordRepoTestSuite) TestDelete() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)

	userID := uuid.New()
	pass := &model.Password{
		UserID:      userID,
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		Checksum:    "testchecksum",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	id, err := passRepo.Create(ctx, pass)
	suite.Require().NoError(err)

	// Удаляем пароль
	err = passRepo.Delete(ctx, id, userID)
	suite.Require().NoError(err)

	// Проверяем, что пароль помечен как удаленный
	passwords, err := passRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)

	for _, p := range passwords {
		if p.ID == id {
			suite.Assert().NotNil(p.DeletedAt, "Password should be marked as deleted")
			break
		}
	}
}

func (suite *PasswordRepoTestSuite) TestDelete_NotFound() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)

	userID := uuid.New()

	err := passRepo.Delete(ctx, 99999, userID)
	suite.Require().Error(err)
	suite.Assert().Contains(err.Error(), "no rows were updated")
}

func (suite *PasswordRepoTestSuite) TestFindByChecksum() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)

	userID := uuid.New()
	checksum := "uniquechecksum123"

	pass := &model.Password{
		UserID:      userID,
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		Checksum:    checksum,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	id, err := passRepo.Create(ctx, pass)
	suite.Require().NoError(err)

	// Ищем по checksum
	found, err := passRepo.FindByChecksum(ctx, userID, checksum)
	suite.Require().NoError(err)
	suite.Require().NotNil(found)
	suite.Assert().Equal(id, found.ID)
	suite.Assert().Equal(checksum, found.Checksum)
}

func (suite *PasswordRepoTestSuite) TestFindByChecksum_NotFound() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)

	userID := uuid.New()

	_, err := passRepo.FindByChecksum(ctx, userID, "nonexistentchecksum")
	suite.Require().Error(err)
}

func (suite *PasswordRepoTestSuite) TestGetUserDeviceLastSinc() {
	ctx := context.Background()
	passRepo := postgres.NewPasswordRepository(suite.db)
	deviceRepo := postgres.NewDeviceRepository(suite.db)

	userID := uuid.New()
	deviceID := uuid.New()

	// Создаем устройство
	device := &model.Device{
		ID:         deviceID,
		UserID:     userID,
		DeviceName: "Test Device",
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}
	err := deviceRepo.Create(ctx, device)
	suite.Require().NoError(err)

	// Создаем пароль
	pass := &model.Password{
		UserID:      userID,
		Login:       "testlogin",
		Password:    "testpassword",
		Description: "test description",
		Checksum:    "testchecksum",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	id, err := passRepo.Create(ctx, pass)
	suite.Require().NoError(err)

	// Получаем пароли с последней синхронизации
	passwords, err := passRepo.GetUserDeviceLastSinc(ctx, userID, deviceID)
	suite.Require().NoError(err)
	suite.Assert().GreaterOrEqual(len(passwords), 1)

	found := false
	for _, p := range passwords {
		if p.ID == id {
			found = true
			break
		}
	}
	suite.Assert().True(found, "Password not found in sync result")
}

func TestPasswordRepoTestSuite(t *testing.T) {
	suite.Run(t, new(PasswordRepoTestSuite))
}
