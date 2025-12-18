package services

import (
	"context"
	"errors"
	"testing"

	"gophKeeper/client/internal/transport"
	models "gophKeeper/pkg/grpchelper"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLocalStorage - мок для LocalStorage
type MockLocalStorage struct {
	mock.Mock
}

func (m *MockLocalStorage) GetPasswords() ([]models.Password, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Password), args.Error(1)
}

func (m *MockLocalStorage) SavePass(pass *models.Password) error {
	args := m.Called(pass)
	return args.Error(0)
}

func (m *MockLocalStorage) UpdatePass(pass *models.Password) error {
	args := m.Called(pass)
	return args.Error(0)
}

func (m *MockLocalStorage) DeletePass(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockLocalStorage) FindPasswordByServerID(serverID int64) (int64, error) {
	args := m.Called(serverID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockLocalStorage) GetUserCredentials() (string, string, string, []byte, error) {
	args := m.Called()
	return args.String(0), args.String(1), args.String(2), args.Get(3).([]byte), args.Error(4)
}

func (m *MockLocalStorage) GetLastSyncTime() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockLocalStorage) SaveLastSyncTime(t int64) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockLocalStorage) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockLocalStorage) Unlock(user, password string) error {
	args := m.Called(user, password)
	return args.Error(0)
}

func (m *MockLocalStorage) LocalRegister(user, password string) error {
	args := m.Called(user, password)
	return args.Error(0)
}

func (m *MockLocalStorage) SaveUserCredentials(login, token, deviceID string, encryptedMasterKey []byte) error {
	args := m.Called(login, token, deviceID, encryptedMasterKey)
	return args.Error(0)
}

func (m *MockLocalStorage) IsLoggedIn() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockLocalStorage) IsFirstRun() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockLocalStorage) SaveCard(card *models.Card) error {
	args := m.Called(card)
	return args.Error(0)
}

func (m *MockLocalStorage) UpdateCard(card *models.Card) error {
	args := m.Called(card)
	return args.Error(0)
}

func (m *MockLocalStorage) GetCards() ([]models.Card, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Card), args.Error(1)
}

func (m *MockLocalStorage) GetCardsByIDs(ids []int64) ([]models.Card, error) {
	args := m.Called(ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Card), args.Error(1)
}

func (m *MockLocalStorage) DeleteCard(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockLocalStorage) SaveText(text *models.TextData) error {
	args := m.Called(text)
	return args.Error(0)
}

func (m *MockLocalStorage) UpdateText(text *models.TextData) error {
	args := m.Called(text)
	return args.Error(0)
}

func (m *MockLocalStorage) GetTexts() ([]models.TextData, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TextData), args.Error(1)
}

func (m *MockLocalStorage) GetTextsByIDs(ids []int64) ([]models.TextData, error) {
	args := m.Called(ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TextData), args.Error(1)
}

func (m *MockLocalStorage) DeleteText(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockLocalStorage) SaveFile(file *models.FileData, content []byte) error {
	args := m.Called(file, content)
	return args.Error(0)
}

func (m *MockLocalStorage) SaveFileMetadata(file *models.FileData) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockLocalStorage) UpdateFile(file *models.FileData) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockLocalStorage) GetFiles() ([]models.FileData, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FileData), args.Error(1)
}

func (m *MockLocalStorage) GetFilesByIDs(ids []int64) ([]models.FileData, error) {
	args := m.Called(ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FileData), args.Error(1)
}

func (m *MockLocalStorage) GetFileByID(id int64) (map[string]interface{}, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLocalStorage) DeleteFileByID(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockTransportInterface - мок для TransportInterface
type MockTransportInterface struct {
	mock.Mock
}

func (m *MockTransportInterface) SyncPasswords(ctx context.Context, localPasswords []models.Password) ([]models.Password, error) {
	args := m.Called(ctx, localPasswords)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Password), args.Error(1)
}

func (m *MockTransportInterface) SyncTexts(ctx context.Context, localTexts []models.TextData) ([]models.TextData, error) {
	args := m.Called(ctx, localTexts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TextData), args.Error(1)
}

func (m *MockTransportInterface) SyncCards(ctx context.Context, localCards []models.Card) ([]models.Card, error) {
	args := m.Called(ctx, localCards)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Card), args.Error(1)
}

func (m *MockTransportInterface) SyncFiles(ctx context.Context, localFiles []models.FileData) ([]models.FileData, error) {
	args := m.Called(ctx, localFiles)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FileData), args.Error(1)
}

func TestSync_Success(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Настраиваем моки
	mockStorage.On("GetLastSyncTime").Return(int64(0), nil)
	mockStorage.On("GetUserCredentials").Return("user", "token", "deviceID", []byte{}, nil)
	mockStorage.On("GetPasswords").Return([]models.Password{}, nil)
	// Контекст будет изменен внутри Sync, используем mock.Anything
	mockTransport.On("SyncPasswords", mock.AnythingOfType("*context.valueCtx"), []models.Password{}).Return([]models.Password{}, nil)
	mockStorage.On("SaveLastSyncTime", mock.AnythingOfType("int64")).Return(nil)

	err := service.Sync(ctx)
	assert.NoError(t, err)

	mockStorage.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestSync_WithCredentialsFromContext(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	ctx := context.Background()

	// Добавляем учетные данные в контекст
	ctx = transport.WithAuthCredentials(ctx, "token", "deviceID")

	// Настраиваем моки
	mockStorage.On("GetLastSyncTime").Return(int64(0), nil)
	mockStorage.On("GetPasswords").Return([]models.Password{}, nil)
	// Контекст будет изменен внутри Sync
	mockTransport.On("SyncPasswords", mock.AnythingOfType("*context.valueCtx"), []models.Password{}).Return([]models.Password{}, nil)
	mockStorage.On("SaveLastSyncTime", mock.AnythingOfType("int64")).Return(nil)

	err := service.Sync(ctx)
	assert.NoError(t, err)

	mockStorage.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestSync_NoCredentials(t *testing.T) {
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

func TestNewSyncService(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)

	service := NewSyncService(mockStorage, &log)
	assert.NotNil(t, service)
	assert.Equal(t, mockStorage, service.storage)
}

func TestNewSyncServiceWithTransport(t *testing.T) {
	log := zerolog.Nop()
	mockStorage := new(MockLocalStorage)
	mockTransport := new(MockTransportInterface)

	service := NewSyncServiceWithTransport(mockStorage, mockTransport, &log)
	assert.NotNil(t, service)
	assert.Equal(t, mockStorage, service.storage)
	assert.Equal(t, mockTransport, service.transport)
}
