package services

import (
	"context"
	"errors"
	"gophKeeper/internal/config"
	"gophKeeper/internal/domain/model"
	"gophKeeper/internal/domain/repository"
	pb "gophKeeper/internal/proto"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Claims определяет структуру данных, которые будут храниться в JWT.
type Claims struct {
	jwt.RegisteredClaims
	UserID   uuid.UUID `json:"user_id"`
	DeviceID string    `json:"device_id"`
}

// Service реализует gRPC сервис KeeperService.
type Service struct {
	pb.AuthServiceServer
	userRepo   repository.UserRepository
	deviceRepo repository.DeviceRepository
	log        zerolog.Logger
	jwtSecret  []byte
}

// NewService создает новый экземпляр сервиса аутентификации.
func NewService(log zerolog.Logger, userRepo repository.UserRepository, deviceRepo repository.DeviceRepository, cfg config.Config) *Service {
	return &Service{
		userRepo:   userRepo,
		deviceRepo: deviceRepo,
		log:        log,
		jwtSecret:  []byte(cfg.HashKey),
	}
}

// Register регистрирует нового пользователя.
func (s *Service) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.log.Info().Str("login", req.GetLogin()).Msg("Registration attempt")

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

		var pgErr *pgconn.PgError
		// Проверяем, является ли ошибка ошибкой уникальности (код 23505 для PostgreSQL)
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			s.log.Info().Str("login", req.GetLogin()).Msg("User already exists, attempting to log in")

			// Пользователь существует, попробуем его найти
			existingUser, findErr := s.userRepo.FindByLogin(ctx, req.GetLogin())
			if findErr != nil {
				s.log.Error().Err(findErr).Msg("Failed to find existing user after unique constraint violation")
				return nil, findErr // Возвращаем ошибку поиска
			}

			// Проверяем пароль
			if err := bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(req.GetPassword())); err != nil {
				s.log.Warn().Str("login", req.GetLogin()).Msg("Invalid password for existing user on registration attempt")
				return nil, status.Error(codes.Unauthenticated, "invalid credentials")
			}

			// Пароль верный, используем существующего пользователя
			user = existingUser
		} else {
			return nil, err // Другая ошибка базы данных
		}
	}
	s.log.Info().Str("user_id", user.ID.String()).Msg("User created successfully")

	// Создаем запись для нового устройства
	device := &model.Device{
		ID:         uuid.New(),
		UserID:     user.ID,
		DeviceName: "Initial Device", // Можно будет дать пользователю возможность переименовать
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		s.log.Error().Err(err).Msg("Failed to create device entry")
		// Здесь нужна логика отката создания пользователя, но пока просто вернем ошибку
		return nil, err
	}
	s.log.Info().Str("device_id", device.ID.String()).Msg("Device registered for user")

	// Генерируем JWT токен
	token, err := s.generateJWT(user.ID, device.ID.String())
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate JWT")
		return nil, err
	}

	return pb.RegisterResponse_builder{
		Token: &token}.Build(), nil
}

// Login аутентифицирует пользователя и возвращает JWT.
func (s *Service) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	s.log.Info().Str("login", req.GetLogin()).Msg("Login attempt")

	// Находим пользователя по логину
	user, err := s.userRepo.FindByLogin(ctx, req.GetLogin())
	if err != nil {
		s.log.Warn().Err(err).Str("login", req.GetLogin()).Msg("Failed to find user during login")
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		s.log.Warn().Str("login", req.GetLogin()).Msg("Invalid password during login")
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// TODO: В будущем здесь можно будет получать ID устройства из запроса или создавать новое.
	// Пока для простоты будем использовать первое найденное или создавать новое.
	// Для данного примера мы просто создадим новое "устройство" при каждом входе.
	device := &model.Device{
		ID:     uuid.New(),
		UserID: user.ID,
	}

	// Генерируем JWT токен
	token, err := s.generateJWT(user.ID, device.ID.String())
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate JWT during login")
		return nil, err
	}

	s.log.Info().Str("login", req.GetLogin()).Msg("User logged in successfully")
	return pb.LoginResponse_builder{Token: &token}.Build(), nil
}

func (s *Service) generateJWT(userID uuid.UUID, deviceID string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Токен живет 24 часа
		},
		UserID:   userID,
		DeviceID: deviceID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
