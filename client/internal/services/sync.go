package services

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/domain"
	"time"

	"gophKeeper/client/internal/transport"

	"github.com/rs/zerolog"
)

// SyncService отвечает за двустороннюю синхронизацию данных с сервером.
type SyncService struct {
	storage      domain.LocalStorage
	transport    TransportInterface // Интерфейс для transport слоя (позволяет мокировать в тестах)
	log          *zerolog.Logger
	lastSyncTime int64 // Время последней успешной синхронизации
}

// NewSyncService создает новый экземпляр сервиса синхронизации.
func NewSyncService(storage domain.LocalStorage, log *zerolog.Logger) *SyncService {
	return &SyncService{
		storage:   storage,
		transport: &defaultTransport{}, // Используем реальный transport по умолчанию
		log:       log,
	}
}

// NewSyncServiceWithTransport создает новый экземпляр сервиса синхронизации с указанным transport.
// Используется для тестирования.
func NewSyncServiceWithTransport(storage domain.LocalStorage, transport TransportInterface, log *zerolog.Logger) *SyncService {
	return &SyncService{
		storage:   storage,
		transport: transport,
		log:       log,
	}
}

// Sync выполняет полную синхронизацию всех типов данных.
func (s *SyncService) Sync(ctx context.Context) error {
	s.log.Info().Msg("Starting full data synchronization")

	var err error
	s.lastSyncTime, err = s.storage.GetLastSyncTime()
	if err != nil {
		s.log.Warn().Err(err).Msg("Could not get last sync time, performing full sync")
		// Если времени нет, используем нулевое время, чтобы синхронизировать все.
		s.lastSyncTime = 0
	}

	tokenFromCtx, deviceIDFromCtx, errFromCtx := transport.GetAuthFromContext(ctx)

	var token, deviceID string
	if errFromCtx != nil || tokenFromCtx == "" || deviceIDFromCtx == "" {
		s.log.Debug().Err(errFromCtx).Msg("No auth credentials in context, getting from storage")
		_, token, deviceID, _, err = s.storage.GetUserCredentials()
		if err != nil {
			return fmt.Errorf("failed to get user credentials: %w", err)
		}
		if token == "" {
			return fmt.Errorf("token is empty, cannot sync")
		}
		if deviceID == "" {
			return fmt.Errorf("deviceID is empty, cannot sync")
		}
		s.log.Debug().Str("token_len", fmt.Sprintf("%d", len(token))).Str("deviceID", deviceID).Msg("Credentials retrieved from storage")
		// Добавляем token и deviceID в контекст
		ctx = transport.WithAuthCredentials(ctx, token, deviceID)
	} else {
		s.log.Debug().Str("token_len", fmt.Sprintf("%d", len(tokenFromCtx))).Str("deviceID", deviceIDFromCtx).Msg("Using auth credentials from context")
		token = tokenFromCtx
		deviceID = deviceIDFromCtx
		// Контекст уже содержит учетные данные, не нужно перезаписывать
	}

	// Синхронизация текстовых заметок
	// if err := s.syncTexts(ctx); err != nil {
	// 	s.log.Error().Err(err).Msg("Text sync failed")
	// }

	// if err := s.syncCards(ctx); err != nil {
	// 	s.log.Error().Err(err).Msg("Card sync failed")
	// }
	if err := s.syncPasswords(ctx); err != nil {
		s.log.Error().Err(err).Msg("Password sync failed")
	}

	if err := s.storage.SaveLastSyncTime(time.Now().Unix()); err != nil {
		s.log.Error().Err(err).Msg("Failed to save last sync time")
		return err
	}

	s.log.Info().Msg("Synchronization completed successfully")
	return nil
}

func (s *SyncService) syncTexts(ctx context.Context) error {
	s.log.Info().Msg("Syncing text notes...")

	s.log.Info().Msg("Text notes sync finished.")
	return nil
}

func (s *SyncService) syncCards(ctx context.Context) error {
	s.log.Info().Msg("Syncing credit cards...")
	s.log.Info().Msg("Credit cards sync finished.")
	return nil
}

func (s *SyncService) syncPasswords(ctx context.Context) error {
	s.log.Info().Msg("Syncing passwords...")
	localPasswords, err := s.storage.GetPasswords()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to get local passwords")
		return err
	}
	s.log.Debug().Int("count", len(localPasswords)).Msg("Found local passwords to sync")

	// 2. Отправляем локальные пароли на сервер и получаем актуальный список.
	// Сервер сам разберется, что создать, обновить или удалить.
	serverPasswords, err := s.transport.SyncPasswords(ctx, localPasswords)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to sync passwords with server")
		return err
	}
	s.log.Debug().Int("count", len(serverPasswords)).Msg("Received passwords from server")

	for _, pass := range serverPasswords {
		// Если запись помечена как удаленная, удаляем ее локально.

		if pass.Deleted {
			var localIDToDelete int64
			if pass.LocalID > 0 {
				localIDToDelete = pass.LocalID
			} else if pass.ServerID > 0 {
				foundID, err := s.storage.FindPasswordByServerID(pass.ServerID)
				if err != nil {
					s.log.Error().Err(err).Int64("server_id", pass.ServerID).Msg("Failed to find password by ServerID for deletion")
					continue
				}
				if foundID == 0 {
					s.log.Debug().Int64("server_id", pass.ServerID).Msg("Password not found locally for deletion, skipping")
					continue
				}
				localIDToDelete = foundID
			} else {
				s.log.Warn().Msg("Received deleted password without both LocalID and ServerID, skipping")
				continue
			}

			if err := s.storage.DeletePass(localIDToDelete); err != nil {
				s.log.Error().Err(err).Int64("local_id", localIDToDelete).Msg("Failed to delete password")
			} else {
				s.log.Debug().Int64("local_id", localIDToDelete).Msg("Deleted password")
			}
			continue
		}

		// Если LocalID пустой, пытаемся найти пароль по ServerID
		if pass.LocalID == 0 {
			if pass.ServerID != 0 {

				// Ищем существующий пароль по ServerID
				localID, err := s.storage.FindPasswordByServerID(pass.ServerID)
				if err != nil {
					s.log.Error().Err(err).Int64("server_id", pass.ServerID).Msg("Failed to find password by ServerID")
					continue
				}
				if localID != 0 {
					pass.LocalID = localID
					if err := s.storage.UpdatePass(&pass); err != nil {
						s.log.Error().Err(err).Int64("local_id", localID).Int64("server_id", pass.ServerID).Msg("Failed to update password from server")
					} else {
						s.log.Debug().Int64("local_id", localID).Int64("server_id", pass.ServerID).Msg("Updated password from server")
					}
				} else {
					if err := s.storage.SavePass(&pass); err != nil {
						s.log.Error().Err(err).Int64("server_id", pass.ServerID).Msg("Failed to save new password from server")
					} else {
						s.log.Debug().Int64("server_id", pass.ServerID).Msg("Saved new password from server")
					}
				}
			} else {
				s.log.Warn().Msg("Received password from server without both LocalID and ServerID, skipping")
			}
		} else {
			// LocalID есть, обновляем существующую запись

			if err := s.storage.UpdatePass(&pass); err != nil {
				s.log.Error().Err(err).Int64("local_id", pass.LocalID).Int64("server_id", pass.ServerID).Msg("Failed to update password from server")
			} else {
				s.log.Debug().Int64("local_id", pass.LocalID).Int64("server_id", pass.ServerID).Msg("Updated password from server")
			}
		}
	}

	s.log.Info().Msg("Passwords sync finished.")
	return nil
}

func (s *SyncService) syncFiles(ctx context.Context) error {
	s.log.Info().Msg("Syncing file metadata...")
	s.log.Info().Msg("File metadata sync finished.")
	return nil
}
