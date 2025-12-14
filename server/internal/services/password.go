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
	log      zerolog.Logger
	passRepo repository.PasswordRepository
}

// NewPasswordService создает новый экземпляр сервиса для паролей.
func NewPasswordService(log zerolog.Logger, passRepo repository.PasswordRepository) *PasswordService {
	return &PasswordService{
		log:      log,
		passRepo: passRepo,
	}
}

// PasswordsSync выполняет полную синхронизацию паролей.
func (s *PasswordService) PasswordsSync(ctx context.Context, req *pb.PasswordsSyncRequest) (*pb.GetPasswordsResponse, error) {
	// Извлекаем userID из контекста (из JWT токена)
	userID, ok := ctx.Value("userID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	s.log.Info().Msgf("Received %d passwords to sync", len(req.GetPasswords()))

	// Конвертируем protobuf пароли в клиентские модели
	clientPasswords := make(map[string]*clientModel.Password)
	for _, pbPass := range req.GetPasswords() {
		pass := clientModel.FromProtoPassword(pbPass)
		// Используем LocalID клиента как ключ
		clientPasswords[pass.LocalID] = &pass
	}

	// Получаем все пароли пользователя из БД
	serverPasswords, err := s.passRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get passwords from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	// Создаем карту серверных паролей по ID
	serverPasswordsMap := make(map[uuid.UUID]*model.Password)
	for _, pass := range serverPasswords {
		serverPasswordsMap[pass.ID] = pass
	}

	var passwordsToClient []*pb.PasswordItem

	// Обрабатываем пароли от клиента
	for localID, clientPass := range clientPasswords {
		// Если пароль удален на клиенте, помечаем его как удаленный на сервере
		if clientPass.Deleted {
			if clientPass.ServerID != "" {
				serverID, err := uuid.Parse(clientPass.ServerID)
				if err == nil {
					if err := s.passRepo.Delete(ctx, serverID, userID); err != nil {
						s.log.Error().Err(err).Str("server_id", serverID.String()).Msg("failed to delete password")
					} else {
						s.log.Info().Str("server_id", serverID.String()).Msg("Password marked as deleted by client")
						// Удаляем из карты, чтобы он не был отправлен клиенту обратно
						delete(serverPasswordsMap, serverID)
					}
				}
			}
			continue
		}

		if clientPass.ServerID == "" {
			// Новый пароль от клиента
			dbPass := &model.Password{
				ID:          uuid.New(),
				UserID:      userID,
				Login:       clientPass.Login,
				Password:    clientPass.Password,
				Description: clientPass.Description,
				Checksum:    clientPass.Checksum,
				CreatedAt:   time.Now().Unix(),
				UpdatedAt:   time.Now().Unix(),
			}

			if err := s.passRepo.Create(ctx, dbPass); err != nil {
				s.log.Error().Err(err).Msgf("failed to create password for local_id %s", localID)
				continue
			}

			// Конвертируем обратно в протобуф для отправки (без шифрования)
			serverIDStr := dbPass.ID.String()
			passwordsToClient = append(passwordsToClient, pb.PasswordItem_builder{
				LocalId:     &localID,
				ServerId:    &serverIDStr,
				Login:       &dbPass.Login,
				Password:    &dbPass.Password,
				Description: &dbPass.Description,
				Checksum:    &dbPass.Checksum,
				Timemap: pb.TimeMap_builder{
					ChangeTime: &dbPass.UpdatedAt,
					SyncTime:   &dbPass.UpdatedAt,
				}.Build(),
				Deleted: &[]bool{false}[0],
			}.Build())
		} else {
			// Существующий пароль
			serverID, err := uuid.Parse(clientPass.ServerID)
			if err != nil {
				s.log.Warn().Str("server_id", clientPass.ServerID).Msg("invalid server_id, skipping")
				continue
			}

			serverPass, found := serverPasswordsMap[serverID]
			if !found {
				// Пароль есть у клиента, но нет на сервере (возможно, был удален на другом устройстве)
				// Помечаем его как удаленный для клиента
				clientPass.Deleted = true
				//passwordsToClient = append(passwordsToClient, clientPass.ToProto())
				continue
			}

			// Сравниваем checksum
			if clientPass.Checksum != serverPass.Checksum {
				// Checksum отличается - сравниваем время изменения
				clientChangeTime := clientPass.ChangeTime
				serverChangeTime := serverPass.UpdatedAt

				if clientChangeTime > serverChangeTime {
					// У клиента версия новее, обновляем на сервере
					serverPass.Login = clientPass.Login
					serverPass.Password = clientPass.Password
					serverPass.Description = clientPass.Description
					serverPass.Checksum = clientPass.Checksum
					serverPass.UpdatedAt = time.Now().Unix()
					if err := s.passRepo.Update(ctx, serverPass); err != nil {
						s.log.Error().Err(err).Msgf("failed to update password with id %s", serverPass.ID)
						continue
					}
				} else {
					// У сервера версия новее, отправляем клиенту
					// responsePass := clientModel.Password{
					// 	LocalID:     localID,
					// 	ServerID:    serverPass.ID.String(),
					// 	Login:       serverPass.Login,
					// 	Password:    serverPass.Password,
					// 	Description: serverPass.Description,
					// 	Checksum:    serverPass.Checksum,
					// 	ChangeTime:  serverPass.UpdatedAt,
					// 	SyncTime:    time.Now().Unix(),
					// 	Deleted:     false,
					// }
					//passwordsToClient = append(passwordsToClient, responsePass.ToProto())
				}
			}
			// Удаляем из карты, чтобы потом найти те, что остались только на сервере
			delete(serverPasswordsMap, serverID)
		}
	}

	// Обрабатываем пароли, которые остались только на сервере
	for _, serverPass := range serverPasswordsMap {
		// Ищем, есть ли у клиента пароль с таким ServerID
		var clientHasIt bool
		for _, clientPass := range clientPasswords {
			if clientPass.ServerID == serverPass.ID.String() {
				clientHasIt = true
				// Если checksum отличается, отправляем полный пароль
				if clientPass.Checksum != serverPass.Checksum {
					// responsePass := clientModel.Password{
					// 	LocalID:     clientPass.LocalID,
					// 	ServerID:    serverPass.ID.String(),
					// 	Login:       serverPass.Login,
					// 	Password:    serverPass.Password,
					// 	Description: serverPass.Description,
					// 	Checksum:    serverPass.Checksum,
					// 	ChangeTime:  serverPass.UpdatedAt,
					// 	SyncTime:    time.Now().Unix(),
					// 	Deleted:     false,
					// }
					//passwordsToClient = append(passwordsToClient, responsePass.ToProto())
				}
				break
			}
		}

		if !clientHasIt {
			// У клиента такого пароля нет, отправляем ему
			// responsePass := clientModel.Password{
			// 	LocalID:     "", // Клиент должен будет присвоить свой local_id
			// 	ServerID:    serverPass.ID.String(),
			// 	Login:       serverPass.Login,
			// 	Password:    serverPass.Password,
			// 	Description: serverPass.Description,
			// 	Checksum:    serverPass.Checksum,
			// 	ChangeTime:  serverPass.UpdatedAt,
			// 	SyncTime:    time.Now().Unix(),
			// 	Deleted:     false,
			// }
			//passwordsToClient = append(passwordsToClient, responsePass.ToProto())
		}
	}

	s.log.Info().Msgf("Sending %d passwords back to client", len(passwordsToClient))
	return pb.GetPasswordsResponse_builder{Passwords: passwordsToClient}.Build(), nil
}

// GetPasswordsByServerIDs получает полные данные паролей по их серверным ID.
func (s *PasswordService) GetPasswordsByServerIDs(ctx context.Context, req *pb.GetPasswordsByServerIDsRequest) (*pb.GetPasswordsResponse, error) {
	// Извлекаем userID из контекста (из JWT токена)
	userID, ok := ctx.Value("userID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	serverIDs := req.GetServerIds()
	s.log.Info().Strs("server_ids", serverIDs).Msgf("Received request to get %d passwords by server IDs", len(serverIDs))

	if len(serverIDs) == 0 {
		return &pb.GetPasswordsResponse{}, nil
	}

	// Получаем пароли из репозитория
	passwords, err := s.passRepo.GetByServerIDs(ctx, userID, serverIDs)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get passwords by server IDs from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	// Конвертируем модели БД в protobuf-модели для ответа
	passwordsToClient := make([]*pb.PasswordItem, 0, len(passwords))
	for _, pass := range passwords {
		// Определяем, удален ли пароль на основе deleted_at
		isDeleted := pass.DeletedAt != nil && *pass.DeletedAt > 0

		serverID := pass.ID.String()
		syncTime := time.Now().Unix()

		passwordsToClient = append(passwordsToClient, pb.PasswordItem_builder{
			ServerId:    &serverID,
			Login:       &pass.Login,
			Password:    &pass.Password,
			Description: &pass.Description,
			Checksum:    &pass.Checksum,
			Timemap: pb.TimeMap_builder{
				ChangeTime: &pass.UpdatedAt,
				SyncTime:   &syncTime,
			}.Build(),
			Deleted: &isDeleted,
		}.Build())
	}

	s.log.Info().Msgf("Sending %d passwords back to client", len(passwordsToClient))
	return pb.GetPasswordsResponse_builder{Passwords: passwordsToClient}.Build(), nil
}
