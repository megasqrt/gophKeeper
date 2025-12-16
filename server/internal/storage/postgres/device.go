package postgres

import (
	"context"
	"database/sql"
	"errors"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// DeviceRepository реализует методы для работы с устройствами в PostgreSQL.
type DeviceRepository struct {
	db *sqlx.DB
}

// NewDeviceRepository создает новый экземпляр DeviceRepository.
func NewDeviceRepository(db *sqlx.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create создает запись о новом устройстве в базе данных.
func (r *DeviceRepository) Create(ctx context.Context, device *model.Device) error {
	query := `INSERT INTO devices (id, user_id, device_name, created_at, updated_at) VALUES (:id, :user_id, :device_name, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, device)
	return err
}

// Update обновляет данные устройства, в основном имя и время обновления.
func (r *DeviceRepository) Update(ctx context.Context, device *model.Device) error {
	query := `UPDATE devices SET device_name = :device_name, updated_at = :updated_at WHERE id = :id`
	result, err := r.db.NamedExecContext(ctx, query, device)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("no rows were updated") // Можно заменить на кастомную ошибку
	}
	return nil
}

func (r *DeviceRepository) SyncTime(ctx context.Context, deviceID uuid.UUID) error {
	query := `UPDATE devices SET last_sync = :last_sync WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, deviceID)
	if err != nil {
		return err
	}
	return nil
}

// FindByUserID находит все устройства, принадлежащие конкретному пользователю.
func (r *DeviceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Device, error) {
	var devices []*model.Device
	query := `SELECT * FROM devices WHERE user_id = $1`
	err := r.db.SelectContext(ctx, &devices, query, userID)
	return devices, err
}

// FindByID находит устройство по его первичному ключу (ID).
func (r *DeviceRepository) FindByID(ctx context.Context, deviceID uuid.UUID) (*model.Device, error) {
	var device model.Device
	query := `SELECT * FROM devices WHERE id = $1`
	err := r.db.GetContext(ctx, &device, query, deviceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("device not found") // Кастомная ошибка
	}
	return &device, err
}

// Delete удаляет устройство по его ID.
func (r *DeviceRepository) Delete(ctx context.Context, deviceID uuid.UUID) error {
	query := `DELETE FROM devices WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, deviceID)
	return err
}
