package postgres_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"gophKeeper/server/internal/domain/model"
	"gophKeeper/server/internal/storage/postgres"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type TextDataRepoTestSuite struct {
	UserRepoTestSuite // Встраиваем UserRepoTestSuite для использования её Setup/TearDown
	user              *model.User
}

func (suite *TextDataRepoTestSuite) SetupSuite() {
	// Вызываем SetupSuite родительской структуры
	suite.UserRepoTestSuite.SetupSuite()

	// Создаем таблицу text_data
	_, err := suite.db.Exec(`
        CREATE TABLE text_data (
            id UUID PRIMARY KEY,
            user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            title VARCHAR(255) NOT NULL,
            text TEXT,
            checksum VARCHAR(64),
            created_at TIMESTAMP WITH TIME ZONE NOT NULL,
            updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
            deleted_at TIMESTAMP WITH TIME ZONE
        );
    `)
	suite.Require().NoError(err)

	// Создаем тестового пользователя
	userRepo := postgres.NewUserRepository(suite.db)
	testUser := &model.User{
		ID:           uuid.New(),
		Login:        "text_user",
		PasswordHash: "some_hash",
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}
	err = userRepo.Create(context.Background(), testUser)
	suite.Require().NoError(err)
	suite.user = testUser
}

func (suite *TextDataRepoTestSuite) TestTextDataRepository() {
	ctx := context.Background()
	repo := postgres.NewTextDataRepository(suite.db)

	var createdData *model.TextData

	suite.Run("Create", func() {
		text := "This is the content of the test note."
		h := sha256.New()
		h.Write([]byte(text))
		data := &model.TextData{
			ID:        uuid.New(),
			UserID:    suite.user.ID,
			Title:     "Test Note",
			Text:      text,
			Checksum:  hex.EncodeToString(h.Sum(nil)),
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		}

		err := repo.Create(ctx, data)
		suite.Require().NoError(err)
		createdData = data
	})

	suite.Run("GetByID", func() {
		suite.Require().NotNil(createdData, "createdData should not be nil")

		found, err := repo.GetByID(ctx, createdData.ID)
		suite.Require().NoError(err)
		suite.Require().NotNil(found)
		suite.Equal(createdData.ID, found.ID)
		suite.Equal(createdData.Title, found.Title)
		suite.Equal(createdData.Text, found.Text)
	})

	suite.Run("GetByUserID", func() {
		all, err := repo.GetByUserID(ctx, suite.user.ID)
		suite.Require().NoError(err)
		suite.Require().NotEmpty(all)
		suite.Equal(1, len(all))
		suite.Equal(createdData.ID, all[0].ID)
	})

	suite.Run("Update", func() {
		suite.Require().NotNil(createdData, "createdData should not be nil")

		createdData.Title = "Updated Title"
		createdData.Text = "Updated content."
		h := sha256.New()
		h.Write([]byte(createdData.Text))
		createdData.Checksum = hex.EncodeToString(h.Sum(nil))
		createdData.UpdatedAt = time.Now().Unix()

		err := repo.Update(ctx, createdData)
		suite.Require().NoError(err)

		updated, err := repo.GetByID(ctx, createdData.ID)
		suite.Require().NoError(err)
		suite.Equal("Updated Title", updated.Title)
		suite.Equal("Updated content.", updated.Text)
	})

	suite.Run("Delete", func() {
		suite.Require().NotNil(createdData, "createdData should not be nil")

		err := repo.Delete(ctx, createdData.ID, suite.user.ID)
		suite.Require().NoError(err)

		// Проверяем, что GetByID не находит удаленную запись
		deleted, err := repo.GetByID(ctx, createdData.ID)
		suite.Require().Error(err)
		suite.Nil(deleted)
		suite.Contains(err.Error(), "text data not found")

		// Проверяем, что GetByUserID также не возвращает ее
		all, err := repo.GetByUserID(ctx, suite.user.ID)
		suite.Require().NoError(err)
		suite.Empty(all)
	})
}

func TestTextDataRepoTestSuite(t *testing.T) {
	suite.Run(t, new(TextDataRepoTestSuite))
}
