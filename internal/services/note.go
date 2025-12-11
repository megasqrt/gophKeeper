package services

import (
	"context"
	"gophKeeper/internal/domain/model"
	"gophKeeper/internal/domain/repository"
	pb "gophKeeper/internal/proto/gen"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// NotesSync выполняет синхронизацию текстовых заметок.
func (s *NoteService) NotesSync(ctx context.Context, req *pb.NotesSyncRequest) (*pb.GetNotesResponse, error) {
	// Извлекаем userID из контекста (из JWT токена)
	userID, ok := ctx.Value("userID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	s.log.Info().Msgf("Received %d notes to sync", len(req.GetNotes()))

	clientNotes := make(map[string]*model.TextData)
	for pbNote := range req.GetNotes() {
		note := model.FromProtoText(pbNote)
		// Используем LocalID клиента как ключ
		clientNotes[note.LocalID] = &note
	}

	// 2. Получаем все заметки пользователя из БД
	serverNotes, err := s.noteRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get notes from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	serverNotesMap := make(map[uuid.UUID]*model.TextData)
	for _, note := range serverNotes {
		serverNotesMap[note.ID] = note
	}

	var notesToClient []*pb.NoteItem

	// 3. Сравниваем версии
	// Проход по заметкам клиента
	for localID, clientNote := range clientNotes {
		if clientNote.ServerID == uuid.Nil {
			// Новая заметка от клиента
			clientNote.ID = uuid.New()
			clientNote.UserID = userID
			clientNote.CreatedAt = time.Now()
			clientNote.UpdatedAt = clientNote.CreatedAt

			if err := s.noteRepo.Create(ctx, clientNote); err != nil {
				s.log.Error().Err(err).Msgf("failed to create note for local_id %s", localID)
				continue
			}
			// Отправляем обратно клиенту с присвоенным ServerID
			notesToClient = append(notesToClient, clientNote.ToProto())
		} else {
			// Существующая заметка
			serverNote, found := serverNotesMap[clientNote.ServerID]
			if !found {
				// Заметка есть у клиента, но нет на сервере (возможно, была удалена на другом устройстве)
				// Помечаем ее как удаленную для клиента
				clientNote.Deleted = true
				notesToClient = append(notesToClient, clientNote.ToProto())
				continue
			}

			// Сравниваем время изменения
			if clientNote.ChangeTime.After(serverNote.UpdatedAt) {
				// У клиента версия новее, обновляем на сервере
				serverNote.Title = clientNote.Title
				serverNote.Text = clientNote.Text
				serverNote.Checksum = clientNote.Checksum
				serverNote.UpdatedAt = time.Now() // Обновляем время на сервере
				if err := s.noteRepo.Update(ctx, serverNote); err != nil {
					s.log.Error().Err(err).Msgf("failed to update note with id %s", serverNote.ID)
				}
			}
			// Удаляем из карты, чтобы потом найти те, что остались только на сервере
			delete(serverNotesMap, clientNote.ServerID)
		}
	}

	// 4. Проход по заметкам, которые остались только на сервере
	// Это значит, что они либо новее, либо их вообще нет у клиента
	for _, serverNote := range serverNotesMap {
		// Ищем, есть ли у клиента заметка с таким ServerID
		var clientHasIt bool
		for _, clientNote := range clientNotes {
			if clientNote.ServerID == serverNote.ID {
				clientHasIt = true
				// Сравниваем чек-суммы. Если они разные, отправляем полный текст.
				if clientNote.Checksum != serverNote.Checksum {
					notesToClient = append(notesToClient, serverNote.ToProto())
				}
				break
			}
		}

		if !clientHasIt {
			// У клиента такой заметки нет, отправляем ему
			notesToClient = append(notesToClient, serverNote.ToProto())
		}
	}

	s.log.Info().Msgf("Sending %d notes back to client", len(notesToClient))
	return &pb.GetNotesResponse{Notes: notesToClient}, nil
}
