package grpc

import (
	"context"
	"gophKeeper/client/internal/config"
	pb "gophKeeper/internal/proto"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client - это централизованный gRPC клиент.
type Client struct {
	Auth pb.AuthServiceClient
	//Cards pb.CardServiceClient
	// Passwords pb.PasswordServiceClient
	// ...
	conn   *grpc.ClientConn
	Status bool
}

// NewClient создает и возвращает новый gRPC клиент.
func NewClient(cfg *config.Config) (*Client, error) {
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
		Auth: pb.NewAuthServiceClient(conn),
		//	Cards: pb.NewCardServiceClient(conn),
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
func (c *Client) Login( login, password string) (*pb.LoginResponse, error) {
	req := &pb.LoginRequest_builder{
		Login:    &login,
		Password: &password,
	}.Build()
	return c.Auth.Login(ctx, req)
}

// Register вызывает RPC-метод Register на сервере.
func (c *Client) Register(ctx context.Context, login, password string) (*pb.RegisterResponse, error) {
	req := &pb.RegisterRequest{
		Login:    login,
		Password: password,
	}
	return c.Auth.Register(ctx, req)
}
