package services

import (
	"context"
	"gophKeeper/internal/domain/model"
	pb "gophKeeper/internal/proto"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository определяет интерфейс для работы с хранилищем пользователей.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByLogin(ctx context.Context, login string) (*model.User, error)
}

// Service реализует gRPC сервис KeeperService.
type Service struct {
	pb.UnimplementedKeeperServiceServer
	userRepo UserRepository
	log      zerolog.Logger
}

// NewService создает новый экземпляр сервиса аутентификации.
func NewService(log zerolog.Logger, userRepo UserRepository) *Service {
	return &Service{
		userRepo: userRepo,
		log:      log,
	}
}

// Register регистрирует нового пользователя.
func (s *Service) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to hash password")
		return nil, err
	}

	user := &model.User{
		ID:           uuid.New(),
		Login:        req.GetLogin(),
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Error().Err(err).Msg("Failed to create user")
		// TODO: Обработать ошибку, если пользователь уже существует.
		return nil, err
	}

	token:="token"

	// TODO: Реализовать генерацию и возврат настоящего JWT токена.
	return pb.RegisterResponse_builder{
		Token: &token}.Build(),
		 nil
}
