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

// NotesShortSync выполняет краткую синхронизацию (только метаданные).
func (s *NoteService) NotesShortSync(ctx context.Context, req *pb.ShortSyncRequest) (*pb.ShortSyncResponse, error) {
	// Извлекаем userID из контекста (из JWT токена)
	userID, ok := ctx.Value("userID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	s.log.Info().Msgf("Received short sync request with %d items", len(req.GetItems()))

	// Получаем все заметки пользователя из БД
	serverNotes, err := s.noteRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get notes from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	// Создаем карту серверных заметок по server_id
	serverNotesMap := make(map[string]*model.TextData)
	for _, note := range serverNotes {
		serverNotesMap[note.ID.String()] = note
	}

	// Определяем, какие заметки нужно синхронизировать полностью
	var localIDsToSync []string

	for _, shortItem := range req.GetItems() {
		localID := shortItem.GetLocalId()
		serverID := shortItem.GetServerId()
		clientChecksum := shortItem.GetChecksum()
		clientDeleted := shortItem.GetDeleted()

		// Если заметка удалена на клиенте, пропускаем
		if clientDeleted {
			continue
		}

		// Если это новая заметка (нет server_id), нужно синхронизировать
		if serverID == "" {
			localIDsToSync = append(localIDsToSync, localID)
			continue
		}

		// Проверяем, есть ли заметка на сервере
		serverNote, found := serverNotesMap[serverID]
		if !found {
			// Заметка есть у клиента, но нет на сервере - нужно синхронизировать
			localIDsToSync = append(localIDsToSync, localID)
			continue
		}

		// Сравниваем checksum
		if serverNote.Checksum != clientChecksum {
			// Checksum отличается - нужно синхронизировать
			localIDsToSync = append(localIDsToSync, localID)
		}
	}

	s.log.Info().Msgf("Returning %d local IDs for full sync", len(localIDsToSync))
	return pb.ShortSyncResponse_builder{LocalIds: localIDsToSync}.Build(), nil
}

// NotesSync выполняет полную синхронизацию текстовых заметок.
func (s *NoteService) NotesSync(ctx context.Context, req *pb.NotesSyncRequest) (*pb.GetNotesResponse, error) {
	// Извлекаем userID из контекста (из JWT токена)
	userID, ok := ctx.Value("userID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	s.log.Info().Msgf("Received %d notes to sync", len(req.GetNotes()))

	// Конвертируем protobuf заметки в клиентские модели
	clientNotes := make(map[string]*clientModel.TextData)
	for _, pbNote := range req.GetNotes() {
		note := clientModel.FromProtoText(pbNote)
		// Используем LocalID клиента как ключ
		clientNotes[note.LocalID] = &note
		s.log.Debug().
			Str("local_id", note.LocalID).
			Str("server_id", note.ServerID).
			Str("title", note.Title).
			Int("text_len", len(note.Text)).
			Str("checksum", note.Checksum).
			Bool("deleted", note.Deleted).
			Msg("Converted note from proto")
	}

	// Получаем все заметки пользователя из БД
	serverNotes, err := s.noteRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get notes from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	// Создаем карту серверных заметок по ID
	serverNotesMap := make(map[uuid.UUID]*model.TextData)
	for _, note := range serverNotes {
		serverNotesMap[note.ID] = note
	}

	var notesToClient []*pb.NoteItem

	// Обрабатываем заметки от клиента
	for localID, clientNote := range clientNotes {
		// Если заметка удалена на клиенте, помечаем ее как удаленную на сервере
		if clientNote.Deleted {
			if clientNote.ServerID != "" {
				serverID, err := uuid.Parse(clientNote.ServerID)
				if err == nil {
					if err := s.noteRepo.Delete(ctx, serverID, userID); err != nil {
						s.log.Error().Err(err).Str("server_id", serverID.String()).Msg("failed to delete note")
					} else {
						s.log.Info().Str("server_id", serverID.String()).Msg("Note marked as deleted by client")
					}
				}
			}
			continue
		}

		if clientNote.ServerID == "" {
			// Новая заметка от клиента
			dbNote := &model.TextData{
				ID:        uuid.New(),
				UserID:    userID,
				Title:     clientNote.Title,
				Text:      clientNote.Text,
				Checksum:  clientNote.Checksum,
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			}

			s.log.Info().
				Str("local_id", localID).
				Str("new_id", dbNote.ID.String()).
				Str("title", dbNote.Title).
				Int("text_len", len(dbNote.Text)).
				Str("checksum", dbNote.Checksum).
				Msg("Creating new note")

			if err := s.noteRepo.Create(ctx, dbNote); err != nil {
				s.log.Error().Err(err).
					Str("local_id", localID).
					Str("note_id", dbNote.ID.String()).
					Msg("failed to create note")
				continue
			}

			s.log.Info().
				Str("local_id", localID).
				Str("note_id", dbNote.ID.String()).
				Msg("Note created successfully")

			// Конвертируем обратно в клиентскую модель для отправки
			responseNote := clientModel.TextData{
				LocalID:    localID,
				ServerID:   dbNote.ID.String(),
				Title:      dbNote.Title,
				Text:       dbNote.Text,
				Checksum:   dbNote.Checksum,
				ChangeTime: dbNote.UpdatedAt,
				SyncTime:   dbNote.UpdatedAt,
				Deleted:    false,
			}
			notesToClient = append(notesToClient, responseNote.ToProto())
		} else {
			// Существующая заметка
			serverID, err := uuid.Parse(clientNote.ServerID)
			if err != nil {
				s.log.Warn().Str("server_id", clientNote.ServerID).Msg("invalid server_id, skipping")
				continue
			}

			serverNote, found := serverNotesMap[serverID]
			if !found {
				// Заметка есть у клиента, но нет на сервере (возможно, была удалена на другом устройстве)
				// Помечаем ее как удаленную для клиента
				clientNote.Deleted = true
				notesToClient = append(notesToClient, clientNote.ToProto())
				continue
			}

			// Сравниваем checksum
			if clientNote.Checksum != serverNote.Checksum {
				// Checksum отличается - сравниваем время изменения
				clientChangeTime := clientNote.ChangeTime
				serverChangeTime := serverNote.UpdatedAt

				if clientChangeTime > serverChangeTime {
					// У клиента версия новее, обновляем на сервере
					serverNote.Title = clientNote.Title
					serverNote.Text = clientNote.Text
					serverNote.Checksum = clientNote.Checksum
					serverNote.UpdatedAt = time.Now().Unix()
					if err := s.noteRepo.Update(ctx, serverNote); err != nil {
						s.log.Error().Err(err).Msgf("failed to update note with id %s", serverNote.ID)
						continue
					}
				} else {
					// У сервера версия новее, отправляем клиенту
					responseNote := clientModel.TextData{
						LocalID:    localID,
						ServerID:   serverNote.ID.String(),
						Title:      serverNote.Title,
						Text:       serverNote.Text,
						Checksum:   serverNote.Checksum,
						ChangeTime: serverNote.UpdatedAt,
						SyncTime:   time.Now().Unix(),
						Deleted:    false,
					}
					notesToClient = append(notesToClient, responseNote.ToProto())
				}
			}
			// Удаляем из карты, чтобы потом найти те, что остались только на сервере
			delete(serverNotesMap, serverID)
		}
	}

	// Обрабатываем заметки, которые остались только на сервере
	for _, serverNote := range serverNotesMap {
		// Ищем, есть ли у клиента заметка с таким ServerID
		var clientHasIt bool
		for _, clientNote := range clientNotes {
			if clientNote.ServerID == serverNote.ID.String() {
				clientHasIt = true
				// Если checksum отличается, отправляем полный текст
				if clientNote.Checksum != serverNote.Checksum {
					responseNote := clientModel.TextData{
						LocalID:    clientNote.LocalID,
						ServerID:   serverNote.ID.String(),
						Title:      serverNote.Title,
						Text:       serverNote.Text,
						Checksum:   serverNote.Checksum,
						ChangeTime: serverNote.UpdatedAt,
						SyncTime:   time.Now().Unix(),
						Deleted:    false,
					}
					notesToClient = append(notesToClient, responseNote.ToProto())
				}
				break
			}
		}

		if !clientHasIt {
			// У клиента такой заметки нет, отправляем ему
			// Генерируем временный local_id (клиент должен будет его использовать)
			responseNote := clientModel.TextData{
				LocalID:    "", // Клиент должен будет присвоить свой local_id
				ServerID:   serverNote.ID.String(),
				Title:      serverNote.Title,
				Text:       serverNote.Text,
				Checksum:   serverNote.Checksum,
				ChangeTime: serverNote.UpdatedAt,
				SyncTime:   time.Now().Unix(),
				Deleted:    false,
			}
			notesToClient = append(notesToClient, responseNote.ToProto())
		}
	}

	s.log.Info().Msgf("Sending %d notes back to client", len(notesToClient))
	return pb.GetNotesResponse_builder{Notes: notesToClient}.Build(), nil
}
