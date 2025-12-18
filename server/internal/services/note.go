package services

import (
	"context"
	//	clientModel "gophKeeper/pkg/grpchelper"
	//	"gophKeeper/server/internal/domain/model"
	pb "gophKeeper/pkg/proto"
	"gophKeeper/server/internal/domain/repository"

	//	"time"

	//	"github.com/google/uuid"
	"github.com/rs/zerolog"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
)

// NoteService реализует gRPC сервис для работы с текстовыми заметками.
type NoteService struct {
	pb.UnimplementedNoteServiceServer
	log      zerolog.Logger
	noteRepo repository.TextDataRepository
}

// NewNoteService создает новый экземпляр сервиса для заметок.
func NewNoteService(log zerolog.Logger, noteRepo repository.TextDataRepository) *NoteService {
	return &NoteService{
		log:      log,
		noteRepo: noteRepo,
	}
}

// NotesSync выполняет полную синхронизацию текстовых заметок.
func (s *NoteService) NotesSync(ctx context.Context, req *pb.NotesSyncRequest) (*pb.GetNotesResponse, error) {
	notesToClient := []*pb.NoteItem{}

	return pb.GetNotesResponse_builder{Notes: notesToClient}.Build(), nil
}
