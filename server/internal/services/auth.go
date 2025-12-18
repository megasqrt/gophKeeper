package services

import (
	"context"
	"errors"
	pb "gophKeeper/pkg/proto"
	"gophKeeper/server/internal/config"
	"gophKeeper/server/internal/domain/model"
	"gophKeeper/server/internal/domain/repository"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Service реализует gRPC сервис KeeperService.
type Service struct {
	pb.AuthServiceServer
	userRepo         repository.UserRepository
	deviceRepo       repository.DeviceRepository
	masterKeyService *MasterKeyService
	jwtService       *JWTService
	log              zerolog.Logger
}

// NewService создает новый экземпляр сервиса аутентификации.
func NewService(log zerolog.Logger, userRepo repository.UserRepository, deviceRepo repository.DeviceRepository, jwtService *JWTService, cfg config.Config) *Service {
	return &Service{
		userRepo:         userRepo,
		deviceRepo:       deviceRepo,
		masterKeyService: NewMasterKeyService(),
		jwtService:       jwtService,
		log:              log,
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

	masterKey, err := s.masterKeyService.GenerateMasterKey()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate master key")
		return nil, err
	}

	encryptedMasterKey, err := s.masterKeyService.EncryptMasterKeyWithPassword(masterKey, req.GetPassword(), []byte(req.GetLogin()))
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to encrypt master key")
		return nil, err
	}

	user := &model.User{
		ID:                 uuid.New(),
		Login:              req.GetLogin(),
		PasswordHash:       string(hashedPassword),
		EncryptedMasterKey: encryptedMasterKey,
		CreatedAt:          time.Now().Unix(),
		UpdatedAt:          time.Now().Unix(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.log.Error().Err(err).Msg("Failed to create user")

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			s.log.Info().Str("login", req.GetLogin()).Msg("User already exists, attempting to log in")

			existingUser, findErr := s.userRepo.FindByLogin(ctx, req.GetLogin())
			if findErr != nil {
				s.log.Error().Err(findErr).Msg("Failed to find existing user after unique constraint violation")
				return nil, findErr // Возвращаем ошибку поиска
			}

			if err := bcrypt.CompareHashAndPassword([]byte(existingUser.PasswordHash), []byte(req.GetPassword())); err != nil {
				s.log.Warn().Str("login", req.GetLogin()).Msg("Invalid password for existing user on registration attempt")
				return nil, status.Error(codes.Unauthenticated, "invalid credentials")
			}

			user = existingUser
		} else {
			return nil, err // Другая ошибка базы данных
		}
	}
	s.log.Info().Str("user_id", user.ID.String()).Msg("User created successfully")

	device := &model.Device{
		ID:         uuid.New(),
		UserID:     user.ID,
		DeviceName: "Initial Device", // Можно будет дать пользователю возможность переименовать
		CreatedAt:  time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		s.log.Error().Err(err).Msg("Failed to create device entry")
		return nil, err
	}
	s.log.Info().Str("device_id", device.ID.String()).Msg("Device registered for user")

	// Генерируем JWT токен
	token, err := s.jwtService.GenerateToken(user.ID, device.ID.String())
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate JWT")
		return nil, err
	}

	return pb.RegisterResponse_builder{
		Token:              &token,
		EncryptedMasterKey: user.EncryptedMasterKey,
	}.Build(), nil
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

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetPassword())); err != nil {
		s.log.Warn().Str("login", req.GetLogin()).Msg("Invalid password during login")
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Если у пользователя еще нет мастер-ключа (старые пользователи), генерируем его
	if len(user.EncryptedMasterKey) == 0 {
		masterKey, err := s.masterKeyService.GenerateMasterKey()
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to generate master key for existing user")
			return nil, err
		}

		encryptedMasterKey, err := s.masterKeyService.EncryptMasterKeyWithPassword(masterKey, req.GetPassword(), []byte(user.Login))
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to encrypt master key for existing user")
			return nil, err
		}

		user.EncryptedMasterKey = encryptedMasterKey
		if err := s.userRepo.Update(ctx, user); err != nil {
			s.log.Error().Err(err).Msg("Failed to save master key for existing user")
			return nil, err
		}
	}

	var device *model.Device
	var deviceIDStr string

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values := md.Get("x-device-id")
		if len(values) > 0 {
			deviceIDStr = values[0]
		}
	}

	if deviceIDStr != "" {
		s.log.Info().Str("deviceID", deviceIDStr).Msg("Found deviceID in context")
		deviceID, err := uuid.Parse(deviceIDStr)
		if err == nil {
			foundDevice, err := s.deviceRepo.FindByID(ctx, deviceID)
			if err == nil && foundDevice.UserID == user.ID {
				s.log.Info().Str("deviceID", deviceIDStr).Msg("Successfully attached to existing device")
				device = foundDevice
			} else {
				s.log.Warn().Err(err).Str("deviceID", deviceIDStr).Msg("Failed to find or verify ownership of device")
			}
		} else {
			s.log.Warn().Err(err).Str("deviceID", deviceIDStr).Msg("Failed to parse deviceID from context")
		}
	}

	if device == nil {
		s.log.Info().Msg("Creating new device for user")
		device = &model.Device{
			ID:         uuid.New(),
			UserID:     user.ID,
			DeviceName: "New Device",
			CreatedAt:  time.Now().Unix(),
			UpdatedAt:  time.Now().Unix(),
		}
		if err := s.deviceRepo.Create(ctx, device); err != nil {
			s.log.Error().Err(err).Msg("Failed to create new device")
			return nil, status.Error(codes.Internal, "failed to register device")
		}
		s.log.Info().Str("deviceID", device.ID.String()).Msg("New device created and saved")
	}

	// Генерируем JWT токен
	token, err := s.jwtService.GenerateToken(user.ID, device.ID.String())
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to generate JWT during login")
		return nil, err
	}

	s.log.Info().Str("login", req.GetLogin()).Msg("User logged in successfully")

	return pb.LoginResponse_builder{
		Token:              &token,
		EncryptedMasterKey: user.EncryptedMasterKey,
		DeviceId:           func(s string) *string { return &s }(device.ID.String()),
	}.Build(), nil
}
