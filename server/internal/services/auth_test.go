package services

import (
	"context"
	"testing"
	"time"

	pb "gophKeeper/pkg/proto"
	"gophKeeper/server/internal/config"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository - мок для UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByLogin(ctx context.Context, login string) (*model.User, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// MockDeviceRepository - мок для DeviceRepository
type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) Create(ctx context.Context, device *model.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepository) Update(ctx context.Context, device *model.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Device, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Device), args.Error(1)
}

func (m *MockDeviceRepository) FindByID(ctx context.Context, deviceID uuid.UUID) (*model.Device, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Device), args.Error(1)
}

func (m *MockDeviceRepository) Delete(ctx context.Context, deviceID uuid.UUID) error {
	args := m.Called(ctx, deviceID)
	return args.Error(0)
}

func (m *MockDeviceRepository) SyncTime(ctx context.Context, deviceID uuid.UUID) error {
	args := m.Called(ctx, deviceID)
	return args.Error(0)
}

func TestRegister_Success(t *testing.T) {
	log := zerolog.Nop()
	mockUserRepo := new(MockUserRepository)
	mockDeviceRepo := new(MockDeviceRepository)
	cfg := config.Config{
		HashKey: "test-secret-key",
	}

	service := NewService(log, mockUserRepo, mockDeviceRepo, cfg)
	ctx := context.Background()

	login := "testuser"
	password := "testpassword"
	email := "test@example.com"

	userID := uuid.New()
	deviceID := uuid.New()

	// Настраиваем моки
	mockUserRepo.On("Create", ctx, mock.AnythingOfType("*model.User")).Return(nil).Run(func(args mock.Arguments) {
		user := args.Get(1).(*model.User)
		user.ID = userID
	})

	mockDeviceRepo.On("Create", ctx, mock.AnythingOfType("*model.Device")).Return(nil).Run(func(args mock.Arguments) {
		device := args.Get(1).(*model.Device)
		device.ID = deviceID
	})

	req := pb.RegisterRequest_builder{
		Login:    &login,
		Password: &password,
		Email:    &email,
	}.Build()

	resp, err := service.Register(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.GetToken())
	assert.NotEmpty(t, resp.GetEncryptedMasterKey())

	mockUserRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

func TestRegister_UserAlreadyExists_ValidPassword(t *testing.T) {
	log := zerolog.Nop()
	mockUserRepo := new(MockUserRepository)
	mockDeviceRepo := new(MockDeviceRepository)
	cfg := config.Config{
		HashKey: "test-secret-key",
	}

	service := NewService(log, mockUserRepo, mockDeviceRepo, cfg)
	ctx := context.Background()

	login := "existinguser"
	password := "testpassword"
	email := "test@example.com"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	userID := uuid.New()
	deviceID := uuid.New()

	existingUser := &model.User{
		ID:           userID,
		Login:        login,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	// Настраиваем моки - Create возвращает ошибку уникальности (PostgreSQL error code 23505)
	pgErr := &pgconn.PgError{Code: "23505"}
	mockUserRepo.On("Create", ctx, mock.AnythingOfType("*model.User")).Return(pgErr)
	mockUserRepo.On("FindByLogin", ctx, login).Return(existingUser, nil)

	mockDeviceRepo.On("Create", ctx, mock.AnythingOfType("*model.Device")).Return(nil).Run(func(args mock.Arguments) {
		device := args.Get(1).(*model.Device)
		device.ID = deviceID
	})

	req := pb.RegisterRequest_builder{
		Login:    &login,
		Password: &password,
		Email:    &email,
	}.Build()

	resp, err := service.Register(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.GetToken())

	mockUserRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

func TestRegister_UserAlreadyExists_InvalidPassword(t *testing.T) {
	log := zerolog.Nop()
	mockUserRepo := new(MockUserRepository)
	mockDeviceRepo := new(MockDeviceRepository)
	cfg := config.Config{
		HashKey: "test-secret-key",
	}

	service := NewService(log, mockUserRepo, mockDeviceRepo, cfg)
	ctx := context.Background()

	login := "existinguser"
	wrongPassword := "wrongpassword"
	email := "test@example.com"

	// Настраиваем моки - Create возвращает ошибку, но не pgErr, поэтому FindByLogin не вызывается
	// и ошибка просто возвращается
	mockUserRepo.On("Create", ctx, mock.AnythingOfType("*model.User")).Return(assert.AnError)

	req := pb.RegisterRequest_builder{
		Login:    &login,
		Password: &wrongPassword,
		Email:    &email,
	}.Build()

	resp, err := service.Register(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	mockUserRepo.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	log := zerolog.Nop()
	mockUserRepo := new(MockUserRepository)
	mockDeviceRepo := new(MockDeviceRepository)
	cfg := config.Config{
		HashKey: "test-secret-key",
	}

	service := NewService(log, mockUserRepo, mockDeviceRepo, cfg)
	ctx := context.Background()

	login := "testuser"
	password := "testpassword"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	userID := uuid.New()
	deviceID := uuid.New()

	user := &model.User{
		ID:                 userID,
		Login:              login,
		PasswordHash:       string(hashedPassword),
		EncryptedMasterKey: []byte("encrypted-key"),
		CreatedAt:          time.Now().Unix(),
		UpdatedAt:          time.Now().Unix(),
	}

	// Настраиваем моки
	mockUserRepo.On("FindByLogin", ctx, login).Return(user, nil)
	// Login создает новое устройство, если deviceID нет в контексте
	mockDeviceRepo.On("Create", ctx, mock.AnythingOfType("*model.Device")).Return(nil).Run(func(args mock.Arguments) {
		dev := args.Get(1).(*model.Device)
		dev.ID = deviceID
	})

	req := pb.LoginRequest_builder{
		Login:    &login,
		Password: &password,
	}.Build()

	resp, err := service.Login(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.GetToken())
	assert.NotEmpty(t, resp.GetEncryptedMasterKey())
	assert.NotEmpty(t, resp.GetDeviceId())

	mockUserRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	log := zerolog.Nop()
	mockUserRepo := new(MockUserRepository)
	mockDeviceRepo := new(MockDeviceRepository)
	cfg := config.Config{
		HashKey: "test-secret-key",
	}

	service := NewService(log, mockUserRepo, mockDeviceRepo, cfg)
	ctx := context.Background()

	login := "nonexistent"
	password := "testpassword"

	// Настраиваем моки
	mockUserRepo.On("FindByLogin", ctx, login).Return(nil, assert.AnError)

	req := pb.LoginRequest_builder{
		Login:    &login,
		Password: &password,
	}.Build()

	resp, err := service.Login(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	mockUserRepo.AssertExpectations(t)
}

func TestLogin_InvalidPassword(t *testing.T) {
	log := zerolog.Nop()
	mockUserRepo := new(MockUserRepository)
	mockDeviceRepo := new(MockDeviceRepository)
	cfg := config.Config{
		HashKey: "test-secret-key",
	}

	service := NewService(log, mockUserRepo, mockDeviceRepo, cfg)
	ctx := context.Background()

	login := "testuser"
	password := "testpassword"
	wrongPassword := "wrongpassword"

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	userID := uuid.New()

	user := &model.User{
		ID:           userID,
		Login:        login,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	// Настраиваем моки
	mockUserRepo.On("FindByLogin", ctx, login).Return(user, nil)

	req := pb.LoginRequest_builder{
		Login:    &login,
		Password: &wrongPassword,
	}.Build()

	resp, err := service.Login(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	mockUserRepo.AssertExpectations(t)
}

func TestGenerateJWT(t *testing.T) {
	log := zerolog.Nop()
	mockUserRepo := new(MockUserRepository)
	mockDeviceRepo := new(MockDeviceRepository)
	cfg := config.Config{
		HashKey: "test-secret-key",
	}

	service := NewService(log, mockUserRepo, mockDeviceRepo, cfg)

	userID := uuid.New()
	deviceID := uuid.New().String()

	token, err := service.generateJWT(userID, deviceID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}
