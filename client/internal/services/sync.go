package services

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/domain"
	"time"

	//model "gophKeeper/pkg/grpchelper"

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

	_, token, deviceID, _, err := s.storage.GetUserCredentials()
	if err != nil {
		return fmt.Errorf("failed to get user credentials: %w", err)
	}

	// Инициализируем Encryptor перед синхронизацией (если еще не инициализирован)
	// Это необходимо для расшифровки данных паролей, полученных с сервера
	// InitializeEncryptor с пустым паролем попытается получить пароль из хранилища
	// Если хранилище разблокировано, пароль будет получен и Encryptor инициализирован
	// if err := s.storage.InitializeEncryptor(""); err != nil {
	// 	s.log.Warn().Err(err).Msg("Failed to initialize Encryptor, some data may not be decryptable")
	// 	// Не возвращаем ошибку, так как это может быть не критично для синхронизации
	// 	// Но логируем предупреждение
	// } else {
	// 	s.log.Debug().Msg("Encryptor initialization completed")
	// }

	// Добавляем token и deviceID в контекст
	ctx = transport.WithAuthCredentials(ctx, token, deviceID)

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
	// if err := s.syncFiles(ctx); err != nil {
	// 	s.log.Error().Err(err).Msg("File metadata sync failed")
	// }

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
	//TODO:
	s.log.Info().Msg("Credit cards sync finished.")
	return nil
}

func (s *SyncService) syncPasswords(ctx context.Context) error {
	s.log.Info().Msg("Syncing passwords...")
	// 1. Получаем все пароли из локального хранилища.
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

	// 3. Обрабатываем ответ от сервера.
	for _, pass := range serverPasswords {
		// Если запись помечена как удаленная, удаляем ее локально.
		if pass.Deleted {
			// Мы можем удалять по LocalID, если он есть, или найти его по ServerID.
			// Для простоты, если LocalID есть, удаляем по нему.
			if pass.LocalID != "" {
				if err := s.storage.DeletePass(pass.LocalID); err != nil {
					s.log.Error().Err(err).Str("local_id", pass.LocalID).Msg("Failed to hard delete password")
				}
			}
			continue
		}

		// Если LocalID пустой, это новая запись с сервера. Сохраняем ее.
		// Если LocalID есть, это обновление существующей записи.
		if err := s.storage.UpdatePass(&pass); err != nil {
			s.log.Error().Err(err).Str("local_id", pass.LocalID).Str("server_id", pass.ServerID).Msg("Failed to save or update password")
		}
	}

	s.log.Info().Msg("Passwords sync finished.")
	return nil
}

func (s *SyncService) syncFiles(ctx context.Context) error {
	s.log.Info().Msg("Syncing file metadata...")
	//TODO:
	s.log.Info().Msg("File metadata sync finished.")
	return nil
}
