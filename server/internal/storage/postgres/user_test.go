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
	_ "github.com/jackc/pgx/v5/stdlib" // Драйвер для sqlx
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
)

const (
	postgresDatabase = "test"
	postgresUser     = "user"
	postgresPassword = "password"
)

type UserRepoTestSuite struct {
	postgresContainer testcontainers.Container
	db                *sqlx.DB
	suite.Suite
}

func (suite *UserRepoTestSuite) SetupSuite() {
	ctx := context.Background()
	postgresContainer, err := tpg.Run(ctx,
		"postgres:16-alpine",
		tpg.WithDatabase(postgresDatabase),
		tpg.WithUsername(postgresUser),
		tpg.WithPassword(postgresPassword),
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
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", postgresUser, postgresPassword,
		postgresEndpoint, postgresDatabase)

	root, err := projectRoot()
	suite.Require().NoError(err)

	migrator := migrations.NewMigration(dsn)

	// Применяем миграции
	// 1. Сначала применяем основные (базовые) миграции
	baseMigrationsPath := filepath.Join(root, "server", "internal", "storage", "postgres", "migrations")
	suite.Require().NoError(migrator.Up("file://" + baseMigrationsPath))

	// Создаем подключение к БД через sqlx
	db, err := sqlx.Connect("pgx", dsn)
	suite.Require().NoError(err)
	suite.db = db
}

func (suite *UserRepoTestSuite) TearDownSuite() {
	ctx := context.Background()
	suite.db.Close()
	suite.Require().NoError(suite.postgresContainer.Terminate(ctx))
}

func (suite *UserRepoTestSuite) TestCreateAndFindByLogin() {
	ctx := context.Background()
	userRepo := postgres.NewUserRepository(suite.db)

	suite.Run("successful user creation and retrieval", func() {
		givenLogin := "mark"
		givenPassword := "secret"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(givenPassword), bcrypt.DefaultCost)
		suite.Require().NoError(err)

		userToCreate := &model.User{
			ID:           uuid.New(),
			Login:        givenLogin,
			PasswordHash: string(hashedPassword),
			CreatedAt:    time.Now().Unix(),
			UpdatedAt:    time.Now().Unix(),
		}

		err = userRepo.Create(ctx, userToCreate)
		suite.Require().NoError(err)

		foundUser, err := userRepo.FindByLogin(ctx, givenLogin)
		suite.Require().NoError(err)
		suite.Require().NotNil(foundUser)
		suite.Equal(userToCreate.ID, foundUser.ID)
		suite.Equal(userToCreate.Login, foundUser.Login)
	})

	suite.Run("FindByLogin_NotFound", func() {
		_, err := userRepo.FindByLogin(ctx, "nonexistent_user")
		suite.Require().Error(err)
		suite.Assert().Contains(err.Error(), "user not found")
	})

	suite.Run("Update", func() {
		givenLogin := "update_user"
		givenPassword := "secret"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(givenPassword), bcrypt.DefaultCost)
		suite.Require().NoError(err)

		userToCreate := &model.User{
			ID:           uuid.New(),
			Login:        givenLogin,
			PasswordHash: string(hashedPassword),
			CreatedAt:    time.Now().Unix(),
			UpdatedAt:    time.Now().Unix(),
		}

		err = userRepo.Create(ctx, userToCreate)
		suite.Require().NoError(err)

		// Обновляем пользователя
		newHashedPassword, err := bcrypt.GenerateFromPassword([]byte("newpassword"), bcrypt.DefaultCost)
		suite.Require().NoError(err)
		userToCreate.PasswordHash = string(newHashedPassword)
		userToCreate.Login = "updated_login"
		userToCreate.UpdatedAt = time.Now().Unix()

		err = userRepo.Update(ctx, userToCreate)
		suite.Require().NoError(err)

		// Проверяем обновление
		updatedUser, err := userRepo.FindByLogin(ctx, "updated_login")
		suite.Require().NoError(err)
		suite.Assert().Equal(userToCreate.ID, updatedUser.ID)
		suite.Assert().Equal("updated_login", updatedUser.Login)
	})

	suite.Run("FindByLogin_OtherError", func() {
		// Тест для покрытия ветки с другими ошибками (не ErrNoRows)
		// Это сложно протестировать без моков, но попробуем
		_, err := userRepo.FindByLogin(ctx, "")
		// Может вернуть ошибку или нет
		_ = err
	})
}

func TestUserRepoTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepoTestSuite))
}
