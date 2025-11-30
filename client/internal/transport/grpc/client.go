package grpc

import (
	"context"
	"errors"
	"gophKeeper/client/internal/config"
	pb "gophKeeper/internal/proto"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var ErrClientNotInitialized = errors.New("gRPC client not initialized")

// Client - это централизованный gRPC клиент.
type Client struct {
	Auth pb.AuthServiceClient
	//Cards pb.CardServiceClient
	// Passwords pb.PasswordServiceClient
	// ...
	conn   *grpc.ClientConn
	Config *config.Config
	Status bool
}

// NewClient создает и возвращает новый gRPC клиент.
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	serverHost := strings.Split(cfg.ServerAddress, ":")[0]
	creds, err := credentials.NewClientTLSFromFile(cfg.CACertPath, serverHost)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(cfg.ServerAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	client := &Client{
		Auth:   pb.NewAuthServiceClient(conn),
		Config: cfg,
		Status: true,
		conn:   conn,
	}

	return client, nil
}

// Close закрывает соединение с сервером.
func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) IsConnected() bool {
	return c.Status
}

// Login вызывает RPC-метод Login на сервере.
func (c *Client) Login(ctx context.Context, login, password string) (*pb.LoginResponse, error) {
	req := pb.LoginRequest_builder{
		Login:    &login,
		Password: &password,
	}.Build()
	return c.Auth.Login(ctx, req)
}

// Register вызывает RPC-метод Register на сервере.
func (c *Client) Register(ctx context.Context, login, password,email string) (*pb.RegisterResponse, error) {
	req := pb.RegisterRequest_builder{
		Login:    &login,
		Password: &password,
		Email:    &email,
	}.Build()
	return c.Auth.Register(ctx, req)
}

func (c *Client) CheckHealth(token string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Добавляем токен в метаданные для аутентификации
	if token != "" {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	}

	conn, err := grpc.NewClient(c.Config.ServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)

	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "", // Проверка всего сервера
	})
	if err != nil {
		return false, err
	}

	return resp.Status == grpc_health_v1.HealthCheckResponse_SERVING, nil
}
