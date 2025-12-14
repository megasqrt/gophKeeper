package services

import (
	"context"
	//	clientModel "gophKeeper/pkg/grpchelper"
	pb "gophKeeper/pkg/proto"
	"gophKeeper/server/internal/domain/repository"

	"github.com/rs/zerolog"

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

// FilesSync выполняет полную синхронизацию файлов.
func (s *FileService) FilesSync(ctx context.Context, req *pb.FilesSyncRequest) (*pb.GetFilesResponse, error) {
	// Извлекаем userID из контекста (из JWT токена)
	// userID, ok := ctx.Value("userID").(uuid.UUID)
	// if !ok {
	// 	return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	// }

	filesToClient := []*pb.FileItem{}

	return pb.GetFilesResponse_builder{Files: filesToClient}.Build(), nil
}
