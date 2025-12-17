package services

import (
	"context"
	"testing"
	"time"

	clientModel "gophKeeper/pkg/grpchelper"
	pb "gophKeeper/pkg/proto"
	"gophKeeper/server/internal/domain/model"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockPasswordRepository - мок для PasswordRepository
type MockPasswordRepository struct {
	mock.Mock
}

func (m *MockPasswordRepository) Create(ctx context.Context, pass *model.Password) (int64, error) {
	args := m.Called(ctx, pass)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPasswordRepository) Update(ctx context.Context, pass *model.Password) error {
	args := m.Called(ctx, pass)
	return args.Error(0)
}

func (m *MockPasswordRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Password, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Password), args.Error(1)
}

func (m *MockPasswordRepository) Delete(ctx context.Context, id int64, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockPasswordRepository) FindByChecksum(ctx context.Context, userID uuid.UUID, checksum string) (*model.Password, error) {
	args := m.Called(ctx, userID, checksum)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Password), args.Error(1)
}

func (m *MockPasswordRepository) GetUserDeviceLastSinc(ctx context.Context, userID uuid.UUID, deviceID uuid.UUID) ([]*model.Password, error) {
	args := m.Called(ctx, userID, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Password), args.Error(1)
}

func TestPasswordsSync_NewPassword(t *testing.T) {
	log := zerolog.Nop()
	mockPassRepo := new(MockPasswordRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	service := NewPasswordService(log, mockPassRepo, mockDeviceRepo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := uuid.New()

	// Добавляем userID и deviceID в контекст
	ctx = context.WithValue(ctx, clientModel.UserKey, userID)
	ctx = context.WithValue(ctx, clientModel.DeviceIDKey, deviceID)

	localID := int64(1)
	login := "testlogin"
	password := "testpassword"
	description := "test description"
	checksum := "testchecksum"
	changeTime := time.Now().Unix()

	// Настраиваем моки
	mockPassRepo.On("GetByUserID", ctx, userID).Return([]*model.Password{}, nil)
	mockPassRepo.On("Create", ctx, mock.AnythingOfType("*model.Password")).Return(int64(100), nil).Run(func(args mock.Arguments) {
		pass := args.Get(1).(*model.Password)
		pass.ID = 100
	})
	mockPassRepo.On("GetUserDeviceLastSinc", ctx, userID, deviceID).Return([]*model.Password{}, nil)
	mockDeviceRepo.On("SyncTime", ctx, deviceID).Return(nil)

	// Создаем запрос с новым паролем (ServerID = 0)
	serverIDZero := int64(0)
	version := int32(1)
	pbPass := pb.PasswordItem_builder{
		LocalId:     &localID,
		ServerId:    &serverIDZero,
		Login:       &login,
		Password:    &password,
		Description: &description,
		Checksum:    &checksum,
		ChangeTime:  &changeTime,
		Version:     &version,
	}.Build()

	req := pb.PasswordsSyncRequest_builder{
		Passwords: []*pb.PasswordItem{pbPass},
	}.Build()

	resp, err := service.PasswordsSync(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.GetPasswords()))

	mockPassRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

func TestPasswordsSync_UpdatePassword(t *testing.T) {
	log := zerolog.Nop()
	mockPassRepo := new(MockPasswordRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	service := NewPasswordService(log, mockPassRepo, mockDeviceRepo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := uuid.New()

	ctx = context.WithValue(ctx, clientModel.UserKey, userID)
	ctx = context.WithValue(ctx, clientModel.DeviceIDKey, deviceID)

	localID := int64(1)
	serverID := int64(100)
	login := "updatedlogin"
	password := "updatedpassword"
	description := "updated description"
	checksum := "updatedchecksum"
	oldChecksum := "oldchecksum"
	changeTime := time.Now().Unix()
	updatedAt := time.Now().Unix() - 100

	// Существующий пароль на сервере
	serverPass := &model.Password{
		ID:          serverID,
		UserID:      userID,
		Login:       "oldlogin",
		Password:    "oldpassword",
		Description: "old description",
		Checksum:    oldChecksum,
		UpdatedAt:   updatedAt,
		Version:     1,
	}

	// Настраиваем моки
	mockPassRepo.On("GetByUserID", ctx, userID).Return([]*model.Password{serverPass}, nil)
	mockPassRepo.On("Update", ctx, mock.AnythingOfType("*model.Password")).Return(nil)
	mockPassRepo.On("GetUserDeviceLastSinc", ctx, userID, deviceID).Return([]*model.Password{}, nil)
	mockDeviceRepo.On("SyncTime", ctx, deviceID).Return(nil)

	// Создаем запрос с обновленным паролем
	version := int32(1)
	pbPass := pb.PasswordItem_builder{
		LocalId:     &localID,
		ServerId:    &serverID,
		Login:       &login,
		Password:    &password,
		Description: &description,
		Checksum:    &checksum,
		ChangeTime:  &changeTime,
		Version:     &version,
	}.Build()

	req := pb.PasswordsSyncRequest_builder{
		Passwords: []*pb.PasswordItem{pbPass},
	}.Build()

	resp, err := service.PasswordsSync(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	mockPassRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

func TestPasswordsSync_DeletePassword(t *testing.T) {
	log := zerolog.Nop()
	mockPassRepo := new(MockPasswordRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	service := NewPasswordService(log, mockPassRepo, mockDeviceRepo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := uuid.New()

	ctx = context.WithValue(ctx, clientModel.UserKey, userID)
	ctx = context.WithValue(ctx, clientModel.DeviceIDKey, deviceID)

	localID := int64(1)
	serverID := int64(100)
	deleted := true

	// Настраиваем моки
	mockPassRepo.On("GetByUserID", ctx, userID).Return([]*model.Password{}, nil)
	mockPassRepo.On("Delete", ctx, serverID, userID).Return(nil)
	mockPassRepo.On("GetUserDeviceLastSinc", ctx, userID, deviceID).Return([]*model.Password{}, nil)
	mockDeviceRepo.On("SyncTime", ctx, deviceID).Return(nil)

	// Создаем запрос с удаленным паролем
	pbPass := pb.PasswordItem_builder{
		LocalId:  &localID,
		ServerId: &serverID,
		Deleted:  &deleted,
	}.Build()

	req := pb.PasswordsSyncRequest_builder{
		Passwords: []*pb.PasswordItem{pbPass},
	}.Build()

	resp, err := service.PasswordsSync(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)

	mockPassRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

func TestPasswordsSync_InvalidUserCredentials(t *testing.T) {
	log := zerolog.Nop()
	mockPassRepo := new(MockPasswordRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	service := NewPasswordService(log, mockPassRepo, mockDeviceRepo)
	ctx := context.Background()

	// Контекст без userID
	req := pb.PasswordsSyncRequest_builder{
		Passwords: []*pb.PasswordItem{},
	}.Build()

	resp, err := service.PasswordsSync(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestPasswordsSync_InvalidDeviceID(t *testing.T) {
	log := zerolog.Nop()
	mockPassRepo := new(MockPasswordRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	service := NewPasswordService(log, mockPassRepo, mockDeviceRepo)
	ctx := context.Background()

	userID := uuid.New()
	ctx = context.WithValue(ctx, clientModel.UserKey, userID)
	// deviceID отсутствует

	req := pb.PasswordsSyncRequest_builder{
		Passwords: []*pb.PasswordItem{},
	}.Build()

	resp, err := service.PasswordsSync(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestPasswordsSync_ServerOnlyPasswords(t *testing.T) {
	log := zerolog.Nop()
	mockPassRepo := new(MockPasswordRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	service := NewPasswordService(log, mockPassRepo, mockDeviceRepo)
	ctx := context.Background()

	userID := uuid.New()
	deviceID := uuid.New()

	ctx = context.WithValue(ctx, clientModel.UserKey, userID)
	ctx = context.WithValue(ctx, clientModel.DeviceIDKey, deviceID)

	// Пароль, который есть только на сервере
	serverOnlyPass := &model.Password{
		ID:          200,
		UserID:      userID,
		Login:       "serverlogin",
		Password:    "serverpassword",
		Description: "server description",
		Checksum:    "serverchecksum",
		UpdatedAt:   time.Now().Unix(),
		Version:     1,
	}

	// Настраиваем моки
	mockPassRepo.On("GetByUserID", ctx, userID).Return([]*model.Password{}, nil)
	mockPassRepo.On("GetUserDeviceLastSinc", ctx, userID, deviceID).Return([]*model.Password{serverOnlyPass}, nil)
	mockDeviceRepo.On("SyncTime", ctx, deviceID).Return(nil)

	req := pb.PasswordsSyncRequest_builder{
		Passwords: []*pb.PasswordItem{},
	}.Build()

	resp, err := service.PasswordsSync(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.GetPasswords()))

	mockPassRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}
