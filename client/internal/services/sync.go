package services

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/domain"
	"time"

	model "gophKeeper/pkg/grpchelper"

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
		// Проверяем тип ошибки для более понятного сообщения
		if err.Error() == "storage is locked, cannot decrypt" ||
			err.Error() == "storage is locked, cannot encrypt" {
			return fmt.Errorf("storage is locked, please unlock with correct password before syncing")
		}
		if err.Error() == "cipher: message authentication failed" ||
			err.Error() == "could not decrypt token: cipher: message authentication failed" {
			return fmt.Errorf("incorrect password: storage was unlocked with wrong password, please unlock with correct password")
		}
		return fmt.Errorf("failed to get user credentials: %w", err)
	}

	// Инициализируем Encryptor перед синхронизацией (если еще не инициализирован)
	// Это необходимо для расшифровки данных паролей, полученных с сервера
	// InitializeEncryptor с пустым паролем попытается получить пароль из хранилища
	// Если хранилище разблокировано, пароль будет получен и Encryptor инициализирован
	if err := s.storage.InitializeEncryptor(""); err != nil {
		s.log.Warn().Err(err).Msg("Failed to initialize Encryptor, some data may not be decryptable")
		// Не возвращаем ошибку, так как это может быть не критично для синхронизации
		// Но логируем предупреждение
	} else {
		s.log.Debug().Msg("Encryptor initialization completed")
	}

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

	// 1. Получаем краткую информацию о текстах
	shortTexts, err := s.storage.GetShortTexts()
	if err != nil {
		return err
	}

	//TODO: remove
	s.log.Debug().Msgf("Found %d short texts to sync", len(shortTexts))

	for i, info := range shortTexts {
		s.log.Debug().Msgf("local [%d] LocalID: %s, ServerID: %s \n", i, info.LocalID, info.ServerID)
	}

	// 2. Отправляем краткую информацию на сервер и получаем ID текстов, которые нужно синхронизировать полностью
	syncResult, err := s.transport.SyncShortTexts(ctx, shortTexts)
	if err != nil {
		return err
	}

	s.log.Debug().Msgf("Short sync result: %d local IDs to send, %d server IDs to receive",
		len(syncResult.LocalIDs), len(syncResult.ServerIDs))

	// 3. Получаем полные данные текстов по LocalID (те, которые нужно отправить на сервер)
	var localTexts []model.TextData

	if len(syncResult.LocalIDs) > 0 {
		localTexts, err = s.storage.GetTextsByIDs(syncResult.LocalIDs)
		if err != nil {
			return err
		}

		// 4. отправляем полные данные текстов на сервер
		serverTexts, err := s.transport.SyncTexts(ctx, localTexts)
		if err != nil {
			return err
		}

		s.log.Debug().Msgf("Received %d texts from server", len(serverTexts))

	}

	if len(syncResult.ServerIDs) > 0 {
		//TODO: Запись новых данных с сервера
	}

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

	

	s.log.Info().Msg("Passwords sync finished.")
	return nil
}

func (s *SyncService) syncFiles(ctx context.Context) error {
	s.log.Info().Msg("Syncing file metadata...")
	//TODO:
	s.log.Info().Msg("File metadata sync finished.")
	return nil
}
