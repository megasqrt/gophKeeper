package services

import (
	"context"
	clientModel "gophKeeper/pkg/grpchelper"
	pb "gophKeeper/pkg/proto"
	"gophKeeper/server/internal/domain/model"
	"gophKeeper/server/internal/domain/repository"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PasswordService реализует gRPC сервис для работы с паролями.
type PasswordService struct {
	pb.UnimplementedPasswordServiceServer
	log        zerolog.Logger
	passRepo   repository.PasswordRepository
	deviceRepo repository.DeviceRepository
}

// NewPasswordService создает новый экземпляр сервиса для паролей.
func NewPasswordService(log zerolog.Logger, passRepo repository.PasswordRepository, deviceRepo repository.DeviceRepository) *PasswordService {
	return &PasswordService{
		log:        log,
		passRepo:   passRepo,
		deviceRepo: deviceRepo,
	}
}

// PasswordsSync выполняет полную синхронизацию паролей.
func (s *PasswordService) PasswordsSync(ctx context.Context, req *pb.PasswordsSyncRequest) (*pb.GetPasswordsResponse, error) {
	userID, ok := ctx.Value(clientModel.UserKey).(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	deviceID, ok := ctx.Value(clientModel.DeviceIDKey).(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid device initialization")
	}

	s.log.Info().Msgf("Received %d passwords to sync", len(req.GetPasswords()))

	clientPasswords := make(map[int64]*clientModel.Password)
	for _, pbPass := range req.GetPasswords() {
		pass := clientModel.FromProtoPassword(pbPass)
		clientPasswords[pass.LocalID] = &pass
	}

	// Получаем все пароли пользователя из БД (и удаленные)
	serverPasswords, err := s.passRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get passwords from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	serverPasswordsMap := make(map[int64]*model.Password)
	for _, pass := range serverPasswords {
		serverPasswordsMap[pass.ID] = pass
	}

	var passwordsToClient []*pb.PasswordItem
	// Карта для отслеживания уже отправленных ServerID, чтобы избежать дубликатов
	sentServerIDs := make(map[int64]bool)

	for localID, clientPass := range clientPasswords {
		if clientPass.Deleted {
			if clientPass.ServerID != 0 {
				if err := s.passRepo.Delete(ctx, clientPass.ServerID, userID); err != nil {
					s.log.Error().Err(err).Int64("server_id", clientPass.ServerID).Msg("failed to delete password")
				} else {
					s.log.Info().Int64("server_id", clientPass.ServerID).Msg("Password marked as deleted by client")
					delete(serverPasswordsMap, clientPass.ServerID)
				}
			}
			continue
		}

		//новый пароль с клиента
		if clientPass.ServerID == 0 {

			dbPass := &model.Password{
				UserID:      userID,
				Login:       clientPass.Login,
				Password:    clientPass.Password,
				Description: clientPass.Description,
				Checksum:    clientPass.Checksum,
				CreatedAt:   time.Now().Unix(),
				UpdatedAt:   time.Now().Unix(),
				Version:     1,
			}

			newServerId, err := s.passRepo.Create(ctx, dbPass)
			if err != nil {
				s.log.Error().Err(err).Msgf("failed to create password for local_id %d", localID)
				continue
			}

			// Конвертируем обратно в протобуф для отправки
			passwordsToClient = append(passwordsToClient, pb.PasswordItem_builder{
				LocalId:     &localID,
				ServerId:    &newServerId,
				Login:       &dbPass.Login,
				Password:    &dbPass.Password,
				Description: &dbPass.Description,
				Checksum:    &dbPass.Checksum,
				ChangeTime:  &dbPass.UpdatedAt,
				Version:     &dbPass.Version,
			}.Build())
			sentServerIDs[newServerId] = true
		} else {
			serverPass, found := serverPasswordsMap[clientPass.ServerID]
			if !found {
				s.log.Error().Msg("Исключение из логики")
				continue
			}

			if clientPass.Checksum != serverPass.Checksum {

				if clientPass.Version < serverPass.Version {
				} else {
					if clientPass.ChangeTime >= serverPass.UpdatedAt {
						serverPass.Login = clientPass.Login
						serverPass.Password = clientPass.Password
						serverPass.Description = clientPass.Description
						serverPass.Checksum = clientPass.Checksum
						serverPass.UpdatedAt = time.Now().Unix()
						serverPass.Version++
						if err := s.passRepo.Update(ctx, serverPass); err != nil {
							s.log.Error().Err(err).Msgf("failed to update password with id %d", serverPass.ID)
							continue
						}
						responsePass := clientModel.Password{
							LocalID:     localID,
							ServerID:    serverPass.ID,
							Login:       serverPass.Login,
							Password:    serverPass.Password,
							Description: serverPass.Description,
							Checksum:    serverPass.Checksum,
							ChangeTime:  serverPass.UpdatedAt,
							Version:     serverPass.Version,
							Deleted:     false,
						}
						passwordsToClient = append(passwordsToClient, responsePass.ToProto())
						sentServerIDs[serverPass.ID] = true
					} else {
						//У сервера версия новее, отправляем клиенту
						responsePass := clientModel.Password{
							LocalID:     localID,
							ServerID:    serverPass.ID,
							Login:       serverPass.Login,
							Password:    serverPass.Password,
							Description: serverPass.Description,
							Checksum:    serverPass.Checksum,
							ChangeTime:  serverPass.UpdatedAt,
							Version:     serverPass.Version,
							Deleted:     false,
						}
						passwordsToClient = append(passwordsToClient, responsePass.ToProto())
						sentServerIDs[serverPass.ID] = true
					}
				}

			}
		}
	}

	newServerPasswords, err := s.passRepo.GetUserDeviceLastSinc(ctx, userID, deviceID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get not sinced passwords from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}
	for _, pass := range newServerPasswords {
		if sentServerIDs[pass.ID] {
			continue
		}

		responsePass := clientModel.Password{
			ServerID:    pass.ID,
			Login:       pass.Login,
			Password:    pass.Password,
			Description: pass.Description,
			Checksum:    pass.Checksum,
			ChangeTime:  pass.UpdatedAt,
			Deleted:     pass.DeletedAt != nil,
			Version:     pass.Version,
		}
		passwordsToClient = append(passwordsToClient, responsePass.ToProto())
		// Отмечаем, что этот ServerID уже отправлен
		sentServerIDs[pass.ID] = true
	}

	s.deviceRepo.SyncTime(ctx, deviceID)

	s.log.Info().Msgf("Sending %d passwords back to client", len(passwordsToClient))
	return pb.GetPasswordsResponse_builder{Passwords: passwordsToClient}.Build(), nil
}
