package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
	
	"gophKeeper/internal/domain/model"
	"gophKeeper/internal/storage/postgres"
	"gophKeeper/pkg/migrations"

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

// projectRoot находит корневую директорию проекта по наличию файла go.mod.
func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if dir == filepath.Dir(dir) {
			return "", errors.New("go.mod not found")
		}
		dir = filepath.Dir(dir)
	}
}
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
	baseMigrationsPath := filepath.Join(root, "internal", "storage", "postgres", "migrations")
	suite.Require().NoError(migrator.Up("file://" + baseMigrationsPath))

	// 2. Затем можем применить тестовые миграции (если они есть)
	// testMigrationsPath := filepath.Join(root, "internal", "storage", "postgres", "migrations_test")
	// suite.Require().NoError(migrator.Up("file://" + testMigrationsPath))

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

	// Закомментированный код аутентификации пока оставим,
	// так как сервис `JWTAuthService` еще не реализован.
	// Мы можем вернуться к нему позже.

	// authService := service.NewJWTAuthService(
	// 	userRepo,
	// 	[]byte("access-secret-key"),
	// 	[]byte("refresh-secret-key"),
	// 	15*time.Minute,
	// 	24*time.Hour,
	// )

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

	// suite.Run("successful token pair generation & refresh", func() {
	// 	tokens, err := authService.GetTokenPair(givenUsername)

	// 	suite.Require().NoError(err)
	// 	suite.NotEmpty(tokens.AccessToken)
	// 	suite.NotEmpty(tokens.RefreshToken)

	// 	actual, err := authService.ValidateAccessToken(tokens.AccessToken)
	// 	suite.Require().NoError(err)
	// 	suite.Equal(givenUsername, actual)

	// 	claims, err := authService.RefreshTokens(tokens.RefreshToken)
	// 	suite.Require().NoError(err)
	// 	actual, err = authService.ValidateAccessToken(claims.AccessToken)
	// 	suite.Require().NoError(err)
	// 	suite.Equal(givenUsername, actual)
	// })

	// suite.Run("successful authentication", func() {
	// 	tokens, err := authService.Authenticate(ctx, givenUsername, givenPassword)

	// 	suite.Require().NoError(err)
	// 	suite.NotEmpty(tokens.AccessToken)
	// 	suite.NotEmpty(tokens.RefreshToken)

	// 	actual, err := authService.ValidateAccessToken(tokens.AccessToken)
	// 	suite.Require().NoError(err)
	// 	suite.Equal(givenUsername, actual)
	// })

	// suite.Run("invalid password", func() {
	// 	_, err := authService.Authenticate(ctx, givenUsername, "geheim")

	// 	suite.Require().Error(err)
	// 	suite.ErrorIs(err, service.ErrInvalidCreds)
	// })

	// suite.Run("invalid user", func() {
	// 	_, err := authService.Authenticate(ctx, "steve", "geheim")

	// 	suite.Require().Error(err)
	// 	suite.ErrorIs(err, service.ErrUserNotFound)
	// })
}

func TestUserRepoTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepoTestSuite))
}
