package postgres_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"gophKeeper/server/internal/domain/model"
	"gophKeeper/server/internal/storage/postgres"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
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

// createTestUser создает тестового пользователя в БД
func createTestUser(db *sqlx.DB, userID uuid.UUID) error {
	userRepo := postgres.NewUserRepository(db)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		ID:           userID,
		Login:        "testuser_" + userID.String()[:8],
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	return userRepo.Create(context.Background(), user)
}
