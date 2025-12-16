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
	deviceRepo repository.DeviceRepository
}

// NewPasswordService создает новый экземпляр сервиса для паролей.
func NewPasswordService(log zerolog.Logger, passRepo repository.PasswordRepository, deviceRepo repository.DeviceRepository) *PasswordService {
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
	deviceID, ok := ctx.Value("deviceID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid device initialization")
	}

	s.log.Info().Msgf("Received %d passwords to sync", len(req.GetPasswords()))

	// Конвертируем protobuf пароли в клиентские модели
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

	// Создаем карту серверных паролей по ID
	serverPasswordsMap := make(map[int64]*model.Password)
	for _, pass := range serverPasswords {
		serverPasswordsMap[pass.ID] = pass
	}

	var passwordsToClient []*pb.PasswordItem



	// Обрабатываем пароли от клиента
	for localID, clientPass := range clientPasswords {
		// Если пароль удален на клиенте, помечаем его как удаленный на сервере
		if clientPass.Deleted {
			if clientPass.ServerID != 0 {
					if err := s.passRepo.Delete(ctx, clientPass.ServerID, userID); err != nil {
						s.log.Error().Err(err).Str("server_id", string(clientPass.ServerID)).Msg("failed to delete password")
					} else {
						s.log.Info().Str("server_id", string(clientPass.ServerID)).Msg("Password marked as deleted by client")
						delete(serverPasswordsMap, clientPass.ServerID)
					}
			}
			continue
		}

		//новый пароль с клиента
		if clientPass.ServerID != 0 {
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

			newServerId,err := s.passRepo.Create(ctx, dbPass); 
			if err != nil {
				s.log.Error().Err(err).Msgf("failed to create password for local_id %s", localID)
				continue
			}
			
			// Конвертируем обратно в протобуф для отправки
			passwordsToClient = append(passwordsToClient, pb.PasswordItem_builder{
				ServerId:    &newServerId,
				Login:       &dbPass.Login,
				Password:    &dbPass.Password,
				Description: &dbPass.Description,
				Checksum:    &dbPass.Checksum,
				ChangeTime: &dbPass.UpdatedAt,
				Version: 	&dbPass.Version,
			}.Build())
			s.log.Debug().Msgf("проверка поведения шифрования %v",passwordsToClient)
		} else {
			// Существующий пароль
			serverPass, found := serverPasswordsMap[clientPass.ServerID]
			if !found {
				// Пароль есть у клиента, но нет на сервере 
				s.log.Error().Msg("Исключение из логики")
				// clientPass.Deleted = true
				// passwordsToClient = append(passwordsToClient, clientPass.ToProto())
				 continue
			}

			// Сравниваем checksum
			if clientPass.Checksum != serverPass.Checksum {
				
				//Cервер доминирует по версии
				if clientPass.Version < serverPass.Version{
					//авто комит 

				} else {
				//версия клиенте равна и надеюсь не больше серверной
					if clientPass.ChangeTime >= serverPass.UpdatedAt {
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
					}else{
						//У сервера версия новее, отправляем клиенту
							responsePass := clientModel.Password{
						LocalID:     localID,
						ServerID:    serverPass.ID,
						Login:       serverPass.Login,
						Password:    serverPass.Password,
						Description: serverPass.Description,
						Checksum:    serverPass.Checksum,
						ChangeTime:  serverPass.UpdatedAt,
						Version: 	 serverPass.Version,
						Deleted:     false,
					}
					passwordsToClient = append(passwordsToClient, responsePass.ToProto())
					}
				}
	
			}
			// Удаляем из карты, чтобы потом найти те, что остались только на сервере
			delete(serverPasswordsMap, clientPass.ServerID)
		}
	}

	// Обрабатываем пароли, которые остались только на сервере
	// Получаем все пароли пользователя из БД (и удаленные)
	newServerPasswords, err := s.passRepo.GetUserDeviceLastSinc(ctx, userID, deviceID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get not sinced passwords from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}
	//отправляем клиенту новые данные с сервера с последней синхронизации
	for _,pass := range newServerPasswords {
		
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
	}

	s.deviceRepo.SyncTime(ctx, deviceID)

	s.log.Info().Msgf("Sending %d passwords back to client", len(passwordsToClient))
	return pb.GetPasswordsResponse_builder{Passwords: passwordsToClient}.Build(), nil
}

