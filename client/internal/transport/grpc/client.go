package grpc

import (
	"context"
	"errors"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/domain/model"
	pb "gophKeeper/internal/proto"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

var ErrClientNotInitialized = errors.New("gRPC client not initialized")

// Client - это централизованный gRPC клиент.
type Client struct {
	Auth pb.AuthServiceClient
	Sync pb.SyncClient
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
		Sync:   pb.NewSyncClient(conn),
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
func (c *Client) Register(ctx context.Context, login, password, email string) (*pb.RegisterResponse, error) {
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

	healthClient := grpc_health_v1.NewHealthClient(c.conn)

	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: "", // Проверка всего сервера
	})
	if err != nil {
		return false, err
	}

	return resp.Status == grpc_health_v1.HealthCheckResponse_SERVING, nil
}

// SyncTexts вызывает RPC для синхронизации текстовых заметок.
func (c *Client) SyncTexts(ctx context.Context, localTexts []model.TextData) ([]model.TextData, error) {
	// Конвертируем наши модели в DTO для gRPC
	pbTexts := make([]*pb.TextData, len(localTexts))
	for i, t := range localTexts {
		pbTexts[i] = t.ToProto()
	}

	textList := pb.TextDataList_builder{
    	Items: pbTexts,
	}.Build()

	req := pb.SyncRequest_builder{
		Texts: textList,
	}.Build()
	resp, err := c.Sync.Sync(ctx, req)
	if err != nil {
		return nil, err
	}

	return model.FromProtoTexts(resp.GetTexts().GetItems()), nil
}

// SyncCards вызывает RPC для синхронизации банковских карт.
func (c *Client) SyncCards(ctx context.Context, localCards []model.Card) ([]model.Card, error) {
	// Конвертируем наши модели в DTO для gRPC
	pbCards := make([]*pb.CardData, len(localCards))
	for i, card := range localCards {
		pbCards[i] = card.ToProto()
	}

	cardsList := pb.CardDataList_builder{
    	Items: pbCards,
	}.Build()

	req := pb.SyncRequest_builder{
		Cards: cardsList,
	}.Build()
	resp, err := c.Sync.Sync(ctx, req)
	if err != nil {
		return nil, err
	}

	// Конвертируем ответ от сервера обратно в наши доменные модели.
	return model.FromProtoCards(resp.GetCards().GetItems()), nil
}

// SyncPasswords вызывает RPC для синхронизации паролей.
func (c *Client) SyncPasswords(ctx context.Context, localPasswords []model.Password) ([]model.Password, error) {
	// Конвертируем наши модели в DTO для gRPC
	pbPasswords := make([]*pb.PasswordData, len(localPasswords))
	for i, pass := range localPasswords {
		pbPasswords[i] = pass.ToProto()
	}

	passwordList := pb.PasswordDataList_builder{
    	Items: pbPasswords,
	}.Build()

	req := pb.SyncRequest_builder{
    	Passwords: passwordList,
	}.Build()
	resp, err := c.Sync.Sync(ctx, req)
	if err != nil {
		return nil, err
	}

	return model.FromProtoPasswords(resp.GetPasswords().GetItems()), nil
}

// SyncFiles вызывает RPC для синхронизации метаданных файлов.
func (c *Client) SyncFiles(ctx context.Context, localFiles []model.FileData) ([]model.FileData, error) {
	// Конвертируем наши модели в DTO для gRPC
	pbFiles := make([]*pb.FileData, len(localFiles))
	for i, file := range localFiles {
		pbFiles[i] = file.ToProto()
	}

	fileList := pb.FileDataList_builder{
		Items: pbFiles,
	}.Build()

	req := pb.SyncRequest_builder{
		Files: fileList,
	}.Build()

	resp, err := c.Sync.Sync(ctx, req)
	if err != nil {
		return nil, err
	}

	return model.FromProtoFiles(resp.GetFiles().GetItems()), nil
}
