package transport

import (
	"context"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/transport/grpc"
	pb "gophKeeper/internal/proto"
)

var (
	// a private singleton instance of the grpc client.
	client   *grpc.Client
	isOnline bool
)

// Init initializes the transport layer. It creates the gRPC client
// and stores it as a package-level singleton.
func Init(ctx context.Context, cfg *config.Config) error {
	var err error
	client, err = grpc.NewClient(ctx, cfg)
	if err != nil {
		isOnline = false
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
		return nil, grpc.ErrClientNotInitialized
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
func Ping(ctx context.Context) error {
	if client == nil {
		isOnline = false
		return grpc.ErrClientNotInitialized
	}
	err := client.Ping(ctx)
	isOnline = err == nil
	return err
}

// IsOnline returns the last known connection status.
func IsOnline() bool {
	return isOnline
}

// Register provides a package-level function for the TUI to call.
// It proxies the call to the underlying gRPC client's method.
func Register(ctx context.Context, login, password string) (*pb.RegisterResponse, error) {
	if client == nil {
		return nil, grpc.ErrClientNotInitialized
	}
	return client.Register(ctx, login, password)
}

