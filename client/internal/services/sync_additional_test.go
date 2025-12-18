package services

import (
	"context"
	"errors"
	"testing"

	models "gophKeeper/pkg/grpchelper"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSync_NoCredentialsFromStorage(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Настраиваем моки
	mockStorage.On("GetLastSyncTime").Return(int64(0), nil)
	mockStorage.On("GetUserCredentials").Return("", "", "", []byte{}, errors.New("no credentials"))

	err := service.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user credentials")

	mockStorage.AssertExpectations(t)
}

func TestSync_EmptyToken(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Настраиваем моки
	mockStorage.On("GetLastSyncTime").Return(int64(0), nil)
	mockStorage.On("GetUserCredentials").Return("user", "", "deviceID", []byte{}, nil)

	err := service.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is empty")

	mockStorage.AssertExpectations(t)
}

func TestSync_EmptyDeviceID(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Настраиваем моки
	mockStorage.On("GetLastSyncTime").Return(int64(0), nil)
	mockStorage.On("GetUserCredentials").Return("user", "token", "", []byte{}, nil)

	err := service.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deviceID is empty")

	mockStorage.AssertExpectations(t)
}

func TestSync_SaveLastSyncTimeError(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Настраиваем моки
	mockStorage.On("GetLastSyncTime").Return(int64(0), nil)
	mockStorage.On("GetUserCredentials").Return("user", "token", "deviceID", []byte{}, nil)
	mockStorage.On("GetPasswords").Return([]models.Password{}, nil)
	mockTransport.On("SyncPasswords", mock.Anything, []models.Password{}).Return([]models.Password{}, nil)
	mockStorage.On("SaveLastSyncTime", mock.AnythingOfType("int64")).Return(errors.New("save error"))

	err := service.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save error")

	mockStorage.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestSyncTexts(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	err := service.syncTexts(ctx)
	assert.NoError(t, err)
}

func TestSyncCards(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	err := service.syncCards(ctx)
	assert.NoError(t, err)
}

func TestSyncFiles(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	err := service.syncFiles(ctx)
	assert.NoError(t, err)
}

func TestSync_GetLastSyncTimeError(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Настраиваем моки
	mockStorage.On("GetLastSyncTime").Return(int64(0), errors.New("db error"))
	mockStorage.On("GetUserCredentials").Return("user", "token", "deviceID", []byte{}, nil)
	mockStorage.On("GetPasswords").Return([]models.Password{}, nil)
	mockTransport.On("SyncPasswords", mock.Anything, []models.Password{}).Return([]models.Password{}, nil)
	mockStorage.On("SaveLastSyncTime", mock.AnythingOfType("int64")).Return(nil)

	err := service.Sync(ctx)
	assert.NoError(t, err) // Ошибка GetLastSyncTime не критична, используется 0

	mockStorage.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestSync_SyncPasswordsError(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Настраиваем моки
	mockStorage.On("GetLastSyncTime").Return(int64(0), nil)
	mockStorage.On("GetUserCredentials").Return("user", "token", "deviceID", []byte{}, nil)
	mockStorage.On("GetPasswords").Return([]models.Password{}, nil)
	mockTransport.On("SyncPasswords", mock.Anything, []models.Password{}).Return(nil, errors.New("sync error"))
	// SaveLastSyncTime вызывается всегда, даже если syncPasswords вернул ошибку (ошибка только логируется)
	mockStorage.On("SaveLastSyncTime", mock.AnythingOfType("int64")).Return(nil)

	err := service.Sync(ctx)
	// По коду sync.go:86-88, ошибка syncPasswords только логируется, но не прерывает выполнение
	assert.NoError(t, err)

	mockStorage.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestSync_GetPasswordsError(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Настраиваем моки
	// GetPasswords возвращает ошибку, которая приводит к ошибке syncPasswords
	// По коду sync.go:86-88, ошибка syncPasswords только логируется, но не прерывает выполнение
	// SaveLastSyncTime вызывается всегда
	mockStorage.On("GetLastSyncTime").Return(int64(0), nil)
	mockStorage.On("GetUserCredentials").Return("user", "token", "deviceID", []byte{}, nil)
	mockStorage.On("GetPasswords").Return(nil, errors.New("db error"))
	mockStorage.On("SaveLastSyncTime", mock.AnythingOfType("int64")).Return(nil)

	err := service.Sync(ctx)
	// Ошибка GetPasswords приводит к ошибке syncPasswords, которая только логируется
	// Sync продолжает выполнение и вызывает SaveLastSyncTime
	assert.NoError(t, err)

	mockStorage.AssertExpectations(t)
}
