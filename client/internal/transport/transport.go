package transport

import (
	"context"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/domain/model"
	"gophKeeper/client/internal/transport/grpc"

	pb "gophKeeper/internal/proto/gen"

	"google.golang.org/grpc/metadata"

	"github.com/rs/zerolog"
)

var (
	// a private singleton instance of the grpc client.
	client   *grpc.Client
	isOnline bool
	log      zerolog.Logger
)

// Init initializes the transport layer. It creates the gRPC client
// and stores it as a package-level singleton.
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

// Login provides a package-level function for the TUI to call.
// It proxies the call to the underlying gRPC client's method.
func Login(ctx context.Context, login, password string) (*pb.LoginResponse, error) {
	if client == nil {
		isOnline = false
		return nil, ErrClientNotInitialized
	}
	// Attempt to login
	res, err := client.Login(ctx, login, password)
	if err != nil {
		isOnline = false
	} else {
		isOnline = true
	}
	return res, err
}

// Ping sends a Ping RPC to the server to check for connectivity and updates the online status.
func Ping(token string) bool {
	if client == nil {
		isOnline = false
		return false
	}

	isHealthy, _ := client.CheckHealth(token)
	isOnline = isHealthy
	return isOnline
}

// IsOnline returns the last known connection status.
func IsOnline() bool {
	return isOnline
}

// Register provides a package-level function for the TUI to call.
// It proxies the call to the underlying gRPC client's method.
func Register(ctx context.Context, login, password, email string) (*pb.RegisterResponse, error) {
	if client == nil {
		return nil, ErrClientNotInitialized
	}
	return client.Register(ctx, login, password, email)
}

// SyncTexts проксирует вызов к gRPC клиенту.
func SyncTexts(ctx context.Context, token, deviceID string, localTexts []model.TextData) ([]model.TextData, error) {
	if client == nil {
		return nil, ErrClientNotInitialized
	}
	ctx = withAuth(ctx, token, deviceID)
	return client.SyncTexts(ctx, localTexts)
}

// SyncCards проксирует вызов к gRPC клиенту.
func SyncCards(ctx context.Context, token, deviceID string, localCards []model.Card) ([]model.Card, error) {
	ctx = withAuth(ctx, token, deviceID)
	return client.SyncCards(ctx, localCards)
}

// SyncPasswords проксирует вызов к gRPC клиенту.
func SyncPasswords(ctx context.Context, token, deviceID string, localPasswords []model.Password) ([]model.Password, error) {
	ctx = withAuth(ctx, token, deviceID)
	return client.SyncPasswords(ctx, localPasswords)
}

// SyncFiles проксирует вызов к gRPC клиенту для синхронизации метаданных файлов.
func SyncFiles(ctx context.Context, token, deviceID string, localFiles []model.FileData) ([]model.FileData, error) {
	if client == nil {
		return nil, ErrClientNotInitialized
	}
	ctx = withAuth(ctx, token, deviceID)
	return client.SyncFiles(ctx, localFiles)
}

// withAuth добавляет необходимые для аутентификации метаданные в контекст.
func withAuth(ctx context.Context, token, deviceID string) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		"authorization", "Bearer "+token,
		"x-device-id", deviceID,
	)
}
