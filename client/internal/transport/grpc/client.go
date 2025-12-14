package grpc

import (
	"context"
	"gophKeeper/client/internal/config"
	model "gophKeeper/pkg/grpchelper"
	pb "gophKeeper/pkg/proto"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

// Client - это централизованный gRPC клиент.
type Client struct {
	Auth     pb.AuthServiceClient     // Сервис аутентификации
	Note     pb.NoteServiceClient     // Сервис для заметок
	Card     pb.CardServiceClient     // Сервис для карт
	Password pb.PasswordServiceClient // Сервис для паролей
	File     pb.FileServiceClient     // Сервис для файлов
	conn     *grpc.ClientConn         // Общее соединение
	Config   *config.Config           // Конфигурация клиента
	Status   bool                     // Статус соединения
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
		Auth:     pb.NewAuthServiceClient(conn),
		Note:     pb.NewNoteServiceClient(conn),
		Card:     pb.NewCardServiceClient(conn),
		Password: pb.NewPasswordServiceClient(conn),
		File:     pb.NewFileServiceClient(conn),
		Config:   cfg,
		Status:   true,
		conn:     conn,
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
	pbNotes := make([]*pb.NoteItem, 0, len(localTexts))
	for _, t := range localTexts {
		if t.Deleted {
			continue
		}
		pbNotes = append(pbNotes, t.ToProto())
	}

	req := pb.NotesSyncRequest_builder{Notes: pbNotes}.Build()

	resp, err := c.Note.NotesSync(ctx, req)
	if err != nil {
		return nil, err
	}

	// Конвертируем ответ от сервера обратно в наши доменные модели.
	syncedTexts := make([]model.TextData, len(resp.GetNotes()))
	for i, pbNote := range resp.GetNotes() {
		syncedTexts[i] = model.FromProtoText(pbNote)
	}
	return syncedTexts, nil
}

// SyncCards вызывает RPC для синхронизации банковских карт.
func (c *Client) SyncCards(ctx context.Context, localCards []model.Card) ([]model.Card, error) {
	// Конвертируем наши модели в DTO для gRPC
	pbCards := make([]*pb.CardItem, len(localCards))
	for i, card := range localCards {
		pbCards[i] = card.ToProto()
	}

	// Предполагаем, что существует CardsSyncRequest и метод CardsSync по аналогии с Notes
	req := pb.CardsSyncRequest_builder{Cards: pbCards}.Build()
	resp, err := c.Card.CardsSync(ctx, req)
	if err != nil {
		return nil, err
	}

	syncedCards := make([]model.Card, len(resp.GetCards()))
	for i, pbCard := range resp.GetCards() {
		syncedCards[i] = model.FromProtoCard(pbCard)
	}
	return syncedCards, nil
}

// SyncPasswords вызывает RPC для синхронизации паролей.
func (c *Client) SyncPasswords(ctx context.Context, localPasswords []model.Password) ([]model.Password, error) {
	// Конвертируем наши модели в DTO для gRPC
	pbPasswords := make([]*pb.PasswordItem, len(localPasswords))
	for i, pass := range localPasswords {
		pbPasswords[i] = pass.ToProto()
	}

	req := pb.PasswordsSyncRequest_builder{Passwords: pbPasswords}.Build()
	resp, err := c.Password.PasswordsSync(ctx, req)
	if err != nil {
		return nil, err
	}

	syncedPasswords := make([]model.Password, len(resp.GetPasswords()))
	for i, pbPass := range resp.GetPasswords() {
		syncedPasswords[i] = model.FromProtoPassword(pbPass)
	}
	return syncedPasswords, nil
}

func (c *Client) GetPasswordsByServerIDs(ctx context.Context, ids []string) ([]model.Password, error) {
	req := pb.GetPasswordsByServerIDsRequest_builder{ServerIds: ids}.Build()
	resp, err := c.Password.GetPasswordsByServerIDs(ctx, req)
	if err != nil {
		return nil, err
	}

	syncedPasswords := make([]model.Password, len(resp.GetPasswords()))
	for i, pbPass := range resp.GetPasswords() {
		syncedPasswords[i] = model.FromProtoPassword(pbPass)
	}
	return syncedPasswords, nil
}

// SyncFiles вызывает RPC для синхронизации метаданных файлов.
func (c *Client) SyncFiles(ctx context.Context, localFiles []model.FileData) ([]model.FileData, error) {
	// Конвертируем наши модели в DTO для gRPC
	pbFiles := make([]*pb.FileItem, len(localFiles))
	for i, file := range localFiles {
		pbFiles[i] = file.ToProto()
	}

	req := pb.FilesSyncRequest_builder{Files: pbFiles}.Build()

	resp, err := c.File.FilesSync(ctx, req)
	if err != nil {
		return nil, err
	}

	syncedFiles := make([]model.FileData, len(resp.GetFiles()))
	for i, pbFile := range resp.GetFiles() {
		syncedFiles[i] = model.FromProtoFile(pbFile)
	}
	return syncedFiles, nil
}
