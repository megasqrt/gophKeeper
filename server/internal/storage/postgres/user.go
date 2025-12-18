package postgres

import (
	"context"
	"database/sql"
	"errors"

	"gophKeeper/server/internal/domain/model"

	"github.com/jmoiron/sqlx"
)

// UserRepository реализует интерфейс repository.UserRepository для PostgreSQL.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository создает новый экземпляр UserRepository.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create создает нового пользователя в базе данных.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, login, password_hash, encrypted_master_key, created_at, updated_at)
		VALUES (:id, :login, :password_hash, :encrypted_master_key, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}

// FindByLogin находит пользователя по его логину.
// Возвращает ErrUserNotFound, если пользователь не найден.
func (r *UserRepository) FindByLogin(ctx context.Context, login string) (*model.User, error) {
	var user model.User
	query := `SELECT id, login, password_hash, encrypted_master_key, created_at, updated_at FROM users WHERE login = $1`

	err := r.db.GetContext(ctx, &user, query, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// Update обновляет данные существующего пользователя.
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET login = :login, password_hash = :password_hash, encrypted_master_key = :encrypted_master_key, updated_at = :updated_at
		WHERE id = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}
