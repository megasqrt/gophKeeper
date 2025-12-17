package transport

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	grpc "gophKeeper/client/internal/transport/grpc"
	model "gophKeeper/pkg/grpchelper"

	pb "gophKeeper/pkg/proto"

	"google.golang.org/grpc/metadata"

	"github.com/rs/zerolog"
)

// WithAuthCredentials добавляет token и deviceID в контекст
func WithAuthCredentials(ctx context.Context, token, deviceID string) context.Context {
	ctx = context.WithValue(ctx, model.TokenKey, token)
	ctx = context.WithValue(ctx, model.DeviceIDKey, deviceID)
	return ctx
}

// GetAuthFromContext извлекает token и deviceID из контекста
func GetAuthFromContext(ctx context.Context) (token, deviceID string, err error) {
	return getAuthFromContext(ctx)
}

// getAuthFromContext извлекает token и deviceID из контекста
func getAuthFromContext(ctx context.Context) (token, deviceID string, err error) {
	tokenVal := ctx.Value(model.TokenKey)
	if tokenVal == nil {
		return "", "", fmt.Errorf("token not found in context")
	}
	token, ok := tokenVal.(string)
	if !ok {
		return "", "", fmt.Errorf("token has invalid type in context")
	}

	deviceIDVal := ctx.Value(model.DeviceIDKey)
	if deviceIDVal == nil {
		return "", "", fmt.Errorf("deviceID not found in context")
	}
	deviceID, ok = deviceIDVal.(string)
	if !ok {
		return "", "", fmt.Errorf("deviceID has invalid type in context")
	}
	return token, deviceID, nil
}

var (
	client   *grpc.Client
	isOnline bool
	log      zerolog.Logger
)

// Init инициализирует транспортный слой. Создает gRPC клиент и сохраняет его как синглтон на уровне пакета.
func Init(ctx context.Context, cfg *config.Config, log *zerolog.Logger) error {
	var err error
	client, err = grpc.NewClient(ctx, cfg)
	if err != nil {
		isOnline = false
		log.Err(err).Msg("Failed to initialize gRPC client")
		return err
	}
	isOnline = true
	return nil
}

// Login предоставляет функцию на уровне пакета для вызова из TUI.
// Проксирует вызов к методу gRPC клиента.
func Login(ctx context.Context, login, password string) (*pb.LoginResponse, error) {
	if client == nil {
		isOnline = false
		return nil, ErrClientNotInitialized
	}
	res, err := client.Login(ctx, login, password)
	if err != nil {
		isOnline = false
	} else {
		isOnline = true
	}
	return res, err
}

// Ping отправляет Ping RPC на сервер для проверки подключения и обновляет статус онлайн.
func Ping(token string) bool {
	if client == nil {
		isOnline = false
		return false
	}

	isHealthy, _ := client.CheckHealth(token)
	isOnline = isHealthy
	return isOnline
}

// IsOnline возвращает последний известный статус подключения.
func IsOnline() bool {
	return isOnline
}

// Register предоставляет функцию на уровне пакета для вызова из TUI.
// Проксирует вызов к методу gRPC клиента.
func Register(ctx context.Context, login, password, email string) (*pb.RegisterResponse, error) {
	if client == nil {
		return nil, ErrClientNotInitialized
	}
	return client.Register(ctx, login, password, email)
}

func SyncTexts(ctx context.Context, localTexts []model.TextData) ([]model.TextData, error) {
	if client == nil {
		return nil, ErrClientNotInitialized
	}
	authedCtx, err := withAuth(ctx)
	if err != nil {
		return nil, err
	}
	return client.SyncTexts(authedCtx, localTexts)
}

func SyncCards(ctx context.Context, localCards []model.Card) ([]model.Card, error) {
	if client == nil {
		return nil, ErrClientNotInitialized
	}
	authedCtx, err := withAuth(ctx)
	if err != nil {
		return nil, err
	}
	return client.SyncCards(authedCtx, localCards)
}

func SyncPasswords(ctx context.Context, localPasswords []model.Password) ([]model.Password, error) {
	if client == nil {
		return nil, ErrClientNotInitialized
	}

	authedCtx, err := withAuth(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to prepare auth context for SyncPasswords")
		return nil, err
	}
	log.Debug().Int("passwords_count", len(localPasswords)).Msg("Calling gRPC SyncPasswords")
	return client.SyncPasswords(authedCtx, localPasswords)
}

// SyncFiles проксирует вызов к gRPC клиенту для синхронизации метаданных файлов.
func SyncFiles(ctx context.Context, localFiles []model.FileData) ([]model.FileData, error) {
	if client == nil {
		return nil, ErrClientNotInitialized
	}
	authedCtx, err := withAuth(ctx)
	if err != nil {
		return nil, err
	}
	return client.SyncFiles(authedCtx, localFiles)
}

func withAuth(ctx context.Context) (context.Context, error) {
	token, deviceID, err := getAuthFromContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get auth from context in withAuth")
		return ctx, fmt.Errorf("failed to get auth from context: %w", err)
	}

	if deviceID == "" {
		log.Error().Msg("deviceID is empty in withAuth")
		return ctx, fmt.Errorf("deviceID is empty, invalid device initialization")
	}

	if token == "" {
		log.Error().Msg("token is empty in withAuth")
		return ctx, fmt.Errorf("token is empty, cannot authenticate")
	}

	log.Debug().Str("deviceID", deviceID).Str("token_len", fmt.Sprintf("%d", len(token))).Msg("Adding auth headers to gRPC context")

	return metadata.AppendToOutgoingContext(ctx,
		"authorization", "Bearer "+token,
		"x-device-id", deviceID,
	), nil
}
