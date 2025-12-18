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

type FileRepoTestSuite struct {
	postgresContainer testcontainers.Container
	db                *sqlx.DB
	suite.Suite
}

func (suite *FileRepoTestSuite) SetupSuite() {
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

func (suite *FileRepoTestSuite) TearDownSuite() {
	ctx := context.Background()
	suite.db.Close()
	suite.Require().NoError(suite.postgresContainer.Terminate(ctx))
}

func (suite *FileRepoTestSuite) SetupTest() {
	// Очищаем таблицы перед каждым тестом
	suite.db.Exec("DELETE FROM binary_data")
	suite.db.Exec("DELETE FROM users")
}

func (suite *FileRepoTestSuite) TestCreate() {
	ctx := context.Background()
	fileRepo := postgres.NewFileRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	file := &model.File{
		UserID:    userID,
		Name:      "test_file.txt",
		Metadata:  "test metadata",
		Size:      1024,
		Checksum:  "testchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := fileRepo.Create(ctx, file)
	suite.Require().NoError(err)
	suite.Assert().NotEqual(int64(0), id)
}

func (suite *FileRepoTestSuite) TestGetByUserID() {
	ctx := context.Background()
	fileRepo := postgres.NewFileRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	// Создаем несколько файлов
	file1 := &model.File{
		UserID:    userID,
		Name:      "file1.txt",
		Metadata:  "metadata1",
		Size:      1024,
		Checksum:  "checksum1",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	file2 := &model.File{
		UserID:    userID,
		Name:      "file2.txt",
		Metadata:  "metadata2",
		Size:      2048,
		Checksum:  "checksum2",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id1, err := fileRepo.Create(ctx, file1)
	suite.Require().NoError(err)

	id2, err := fileRepo.Create(ctx, file2)
	suite.Require().NoError(err)

	files, err := fileRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)
	suite.Assert().GreaterOrEqual(len(files), 2)

	found1 := false
	found2 := false
	for _, f := range files {
		if f.ID == id1 {
			found1 = true
			suite.Assert().Equal("file1.txt", f.Name)
		}
		if f.ID == id2 {
			found2 = true
			suite.Assert().Equal("file2.txt", f.Name)
		}
	}
	suite.Assert().True(found1, "File 1 not found")
	suite.Assert().True(found2, "File 2 not found")
}

func (suite *FileRepoTestSuite) TestUpdate() {
	ctx := context.Background()
	fileRepo := postgres.NewFileRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	file := &model.File{
		UserID:    userID,
		Name:      "original_file.txt",
		Metadata:  "original metadata",
		Size:      1024,
		Checksum:  "originalchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := fileRepo.Create(ctx, file)
	suite.Require().NoError(err)

	// Обновляем файл
	file.ID = id
	file.Name = "updated_file.txt"
	file.Metadata = "updated metadata"
	file.Size = 2048
	file.Checksum = "updatedchecksum"
	file.UpdatedAt = time.Now().Unix()

	err = fileRepo.Update(ctx, file)
	suite.Require().NoError(err)

	// Проверяем обновление
	files, err := fileRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)

	found := false
	for _, f := range files {
		if f.ID == id {
			found = true
			suite.Assert().Equal("updated_file.txt", f.Name)
			suite.Assert().Equal("updated metadata", f.Metadata)
			suite.Assert().Equal(int64(2048), f.Size)
			break
		}
	}
	suite.Assert().True(found, "Updated file not found")
}

func (suite *FileRepoTestSuite) TestUpdate_NotFound() {
	ctx := context.Background()
	fileRepo := postgres.NewFileRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	file := &model.File{
		ID:        99999,
		UserID:    userID,
		Name:      "nonexistent.txt",
		Metadata:  "metadata",
		Size:      1024,
		Checksum:  "checksum",
		UpdatedAt: time.Now().Unix(),
	}

	err = fileRepo.Update(ctx, file)
	suite.Require().Error(err)
	suite.Assert().Contains(err.Error(), "no rows were updated")
}

func (suite *FileRepoTestSuite) TestDelete() {
	ctx := context.Background()
	fileRepo := postgres.NewFileRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	file := &model.File{
		UserID:    userID,
		Name:      "test_file.txt",
		Metadata:  "test metadata",
		Size:      1024,
		Checksum:  "testchecksum",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Version:   1,
	}

	id, err := fileRepo.Create(ctx, file)
	suite.Require().NoError(err)

	// Удаляем файл
	err = fileRepo.Delete(ctx, id, userID)
	suite.Require().NoError(err)

	// Проверяем, что файл не возвращается в GetByUserID
	files, err := fileRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)
	found := false
	for _, f := range files {
		if f.ID == id {
			found = true
			break
		}
	}
	suite.Assert().False(found, "Deleted file should not be returned")
}

func (suite *FileRepoTestSuite) TestGetByUserID_Empty() {
	ctx := context.Background()
	fileRepo := postgres.NewFileRepository(suite.db)

	userID := uuid.New()
	err := createTestUser(suite.db, userID)
	suite.Require().NoError(err)

	files, err := fileRepo.GetByUserID(ctx, userID)
	suite.Require().NoError(err)
	suite.Assert().Empty(files)
}

func TestFileRepoTestSuite(t *testing.T) {
	suite.Run(t, new(FileRepoTestSuite))
}
