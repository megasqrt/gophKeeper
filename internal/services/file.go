package services

import (
	"context"
	"gophKeeper/internal/domain/model"
	"gophKeeper/internal/domain/repository"
	pb "gophKeeper/internal/proto/gen"
	clientModel "gophKeeper/pkg/grpchelper"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FileService реализует gRPC сервис для работы с файлами.
type FileService struct {
	pb.UnimplementedFileServiceServer
	log      zerolog.Logger
	fileRepo repository.FileRepository
}

// NewFileService создает новый экземпляр сервиса для файлов.
func NewFileService(log zerolog.Logger, fileRepo repository.FileRepository) *FileService {
	return &FileService{
		log:      log,
		fileRepo: fileRepo,
	}
}

// FilesShortSync выполняет краткую синхронизацию (только метаданные).
func (s *FileService) FilesShortSync(ctx context.Context, req *pb.ShortSyncRequest) (*pb.ShortSyncResponse, error) {
	// Извлекаем userID из контекста (из JWT токена)
	userID, ok := ctx.Value("userID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	s.log.Info().Msgf("Received short sync request with %d items", len(req.GetItems()))

	// Получаем все файлы пользователя из БД
	serverFiles, err := s.fileRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get files from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	// Создаем карту серверных файлов по server_id
	serverFilesMap := make(map[string]*model.File)
	for _, file := range serverFiles {
		serverFilesMap[file.ID.String()] = file
	}

	// Определяем, какие файлы нужно синхронизировать полностью
	var localIDsToSync []string

	for _, shortItem := range req.GetItems() {
		localID := shortItem.GetLocalId()
		serverID := shortItem.GetServerId()
		clientChecksum := shortItem.GetChecksum()

		// Если это новый файл (нет server_id), нужно синхронизировать
		if serverID == "" {
			localIDsToSync = append(localIDsToSync, localID)
			continue
		}

		// Проверяем, есть ли файл на сервере
		serverFile, found := serverFilesMap[serverID]
		if !found {
			// Файл есть у клиента, но нет на сервере - нужно синхронизировать
			localIDsToSync = append(localIDsToSync, localID)
			continue
		}

		// Сравниваем checksum
		if serverFile.Checksum != clientChecksum {
			// Checksum отличается - нужно синхронизировать
			localIDsToSync = append(localIDsToSync, localID)
		}
	}

	s.log.Info().Msgf("Returning %d local IDs for full sync", len(localIDsToSync))
	return pb.ShortSyncResponse_builder{LocalIds: localIDsToSync}.Build(), nil
}

// FilesSync выполняет полную синхронизацию файлов.
func (s *FileService) FilesSync(ctx context.Context, req *pb.FilesSyncRequest) (*pb.GetFilesResponse, error) {
	// Извлекаем userID из контекста (из JWT токена)
	userID, ok := ctx.Value("userID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	s.log.Info().Msgf("Received %d files to sync", len(req.GetFiles()))

	// Конвертируем protobuf файлы в клиентские модели
	clientFiles := make(map[string]*clientModel.FileData)
	for _, pbFile := range req.GetFiles() {
		file := clientModel.FromProtoFile(pbFile)
		// Используем LocalID клиента как ключ
		clientFiles[file.LocalID] = &file
	}

	// Получаем все файлы пользователя из БД
	serverFiles, err := s.fileRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get files from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	// Создаем карту серверных файлов по ID
	serverFilesMap := make(map[uuid.UUID]*model.File)
	for _, file := range serverFiles {
		serverFilesMap[file.ID] = file
	}

	var filesToClient []*pb.FileItem

	// Обрабатываем файлы от клиента
	for localID, clientFile := range clientFiles {
		// Если файл удален на клиенте, помечаем его как удаленный на сервере
		if clientFile.Deleted {
			if clientFile.ServerID != "" {
				serverID, err := uuid.Parse(clientFile.ServerID)
				if err == nil {
					if err := s.fileRepo.Delete(ctx, serverID, userID); err != nil {
						s.log.Error().Err(err).Str("server_id", serverID.String()).Msg("failed to delete file")
					} else {
						s.log.Info().Str("server_id", serverID.String()).Msg("File marked as deleted by client")
					}
				}
			}
			continue
		}

		if clientFile.ServerID == "" {
			// Новый файл от клиента
			dbFile := &model.File{
				ID:        uuid.New(),
				UserID:    userID,
				Name:      clientFile.Name,
				Metadata:  clientFile.Metadata,
				Size:      clientFile.Size,
				Checksum:  clientFile.Checksum,
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			}

			if err := s.fileRepo.Create(ctx, dbFile); err != nil {
				s.log.Error().Err(err).Msgf("failed to create file for local_id %s", localID)
				continue
			}

			// Конвертируем обратно в клиентскую модель для отправки
			responseFile := clientModel.FileData{
				LocalID:    localID,
				ServerID:   dbFile.ID.String(),
				Name:       dbFile.Name,
				Metadata:   dbFile.Metadata,
				Size:       dbFile.Size,
				Checksum:   dbFile.Checksum,
				ChangeTime: dbFile.UpdatedAt,
				SyncTime:   dbFile.UpdatedAt,
				Deleted:    false,
			}
			filesToClient = append(filesToClient, responseFile.ToProto())
		} else {
			// Существующий файл
			serverID, err := uuid.Parse(clientFile.ServerID)
			if err != nil {
				s.log.Warn().Str("server_id", clientFile.ServerID).Msg("invalid server_id, skipping")
				continue
			}

			serverFile, found := serverFilesMap[serverID]
			if !found {
				// Файл есть у клиента, но нет на сервере (возможно, был удален на другом устройстве)
				// Помечаем его как удаленный для клиента
				clientFile.Deleted = true
				filesToClient = append(filesToClient, clientFile.ToProto())
				continue
			}

			// Сравниваем checksum
			if clientFile.Checksum != serverFile.Checksum {
				// Checksum отличается - сравниваем время изменения
				clientChangeTime := clientFile.ChangeTime
				serverChangeTime := serverFile.UpdatedAt

				if clientChangeTime > serverChangeTime {
					// У клиента версия новее, обновляем на сервере
					serverFile.Name = clientFile.Name
					serverFile.Metadata = clientFile.Metadata
					serverFile.Size = clientFile.Size
					serverFile.Checksum = clientFile.Checksum
					serverFile.UpdatedAt = time.Now().Unix()
					if err := s.fileRepo.Update(ctx, serverFile); err != nil {
						s.log.Error().Err(err).Msgf("failed to update file with id %s", serverFile.ID)
						continue
					}
				} else {
					// У сервера версия новее, отправляем клиенту
					responseFile := clientModel.FileData{
						LocalID:    localID,
						ServerID:   serverFile.ID.String(),
						Name:       serverFile.Name,
						Metadata:   serverFile.Metadata,
						Size:       serverFile.Size,
						Checksum:   serverFile.Checksum,
						ChangeTime: serverFile.UpdatedAt,
						SyncTime:   time.Now().Unix(),
						Deleted:    false,
					}
					filesToClient = append(filesToClient, responseFile.ToProto())
				}
			}
			// Удаляем из карты, чтобы потом найти те, что остались только на сервере
			delete(serverFilesMap, serverID)
		}
	}

	// Обрабатываем файлы, которые остались только на сервере
	for _, serverFile := range serverFilesMap {
		// Ищем, есть ли у клиента файл с таким ServerID
		var clientHasIt bool
		for _, clientFile := range clientFiles {
			if clientFile.ServerID == serverFile.ID.String() {
				clientHasIt = true
				// Если checksum отличается, отправляем полный файл
				if clientFile.Checksum != serverFile.Checksum {
					responseFile := clientModel.FileData{
						LocalID:    clientFile.LocalID,
						ServerID:   serverFile.ID.String(),
						Name:       serverFile.Name,
						Metadata:   serverFile.Metadata,
						Size:       serverFile.Size,
						Checksum:   serverFile.Checksum,
						ChangeTime: serverFile.UpdatedAt,
						SyncTime:   time.Now().Unix(),
						Deleted:    false,
					}
					filesToClient = append(filesToClient, responseFile.ToProto())
				}
				break
			}
		}

		if !clientHasIt {
			// У клиента такого файла нет, отправляем ему
			responseFile := clientModel.FileData{
				LocalID:    "", // Клиент должен будет присвоить свой local_id
				ServerID:   serverFile.ID.String(),
				Name:       serverFile.Name,
				Metadata:   serverFile.Metadata,
				Size:       serverFile.Size,
				Checksum:   serverFile.Checksum,
				ChangeTime: serverFile.UpdatedAt,
				SyncTime:   time.Now().Unix(),
				Deleted:    false,
			}
			filesToClient = append(filesToClient, responseFile.ToProto())
		}
	}

	s.log.Info().Msgf("Sending %d files back to client", len(filesToClient))
	return pb.GetFilesResponse_builder{Files: filesToClient}.Build(), nil
}
