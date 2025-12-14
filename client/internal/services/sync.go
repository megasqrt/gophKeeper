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

	// Получаем учетные данные для запросов
	// GetUserCredentials использует s.key для расшифровки
	// Если s.key == nil, вернется ошибка "storage is locked"
	// Если s.key неправильный, вернется ошибка "cipher: message authentication failed"
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

	// 1. Получаем краткую информацию о картах
	shortCards, err := s.storage.GetShortCards()
	if err != nil {
		return err
	}

	// 2. Отправляем краткую информацию на сервер и получаем ID карт, которые нужно синхронизировать полностью
	syncResult, err := s.transport.SyncShortCards(ctx, shortCards)
	if err != nil {
		return err
	}

	s.log.Debug().Msgf("Short sync result: %d local IDs to send, %d server IDs to receive",
		len(syncResult.LocalIDs), len(syncResult.ServerIDs))

	// 3. Получаем полные данные карт по LocalID (те, которые нужно отправить на сервер)
	var localCards []model.Card
	var activeLocalCards []model.Card

	if len(syncResult.LocalIDs) > 0 {
		localCards, err = s.storage.GetCardsByIDs(syncResult.LocalIDs)
		if err != nil {
			return err
		}

		// Фильтруем удаленные карты перед отправкой на сервер
		activeLocalCards = make([]model.Card, 0, len(localCards))
		for _, card := range localCards {
			if card.Deleted {
				continue
			}
			card.SyncTime = s.lastSyncTime
			activeLocalCards = append(activeLocalCards, card)
		}
		s.log.Debug().Msgf("Prepared %d active cards to send to server", len(activeLocalCards))
	}

	// 4. ВСЕГДА отправляем полные данные карт на сервер (даже если список пустой)
	// Сервер всегда возвращает все данные, которые нужно получить клиенту
	serverCards, err := s.transport.SyncCards(ctx, activeLocalCards)
	if err != nil {
		return err
	}

	// 5. Обрабатываем ответ сервера
	localSyncables := make([]domain.Syncable, len(localCards))
	for i := range localCards {
		localSyncables[i] = &localCards[i]
	}
	serverSyncables := make([]domain.Syncable, len(serverCards))
	for i := range serverCards {
		serverSyncables[i] = &serverCards[i]
	}

	// saveCardFunc := func(data map[string]interface{}) error {
	// 	card, err := model.FromMapCard(data)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to convert map to card model on save: %w", err)
	// 	}
	// 	// Если в данных уже есть LocalID, используем UpdateCard вместо SaveCard
	// 	if card.LocalID != "" {
	// 		s.log.Warn().Str("local_id", card.LocalID).Msg("Card has LocalID in saveFunc, using UpdateCard instead")
	// 		return s.storage.UpdateCard(&card)
	// 	}
	// 	return s.storage.SaveCard(&card)
	// }
	// updateCardFunc := func(data map[string]interface{}) error {
	// 	card, err := model.FromMapCard(data)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to convert map to card model on update: %w", err)
	// 	}
	// 	return s.storage.UpdateCard(&card)
	// }

	// s.processSyncResults(localSyncables, serverSyncables, saveCardFunc, updateCardFunc,
	// 	s.storage.DeleteCard,
	// 	"card",
	// )

	s.log.Info().Msg("Credit cards sync finished.")
	return nil
}

func (s *SyncService) syncPasswords(ctx context.Context) error {
	s.log.Info().Msg("Syncing passwords...")

	// 1. Получаем краткую информацию о паролях которые нужно синхронизировать
	shortPasswords, err := s.storage.GetShortPasswords()
	if err != nil {
		return err
	}

	// 2. Отправляем краткую информацию на сервер и получаем ID паролей, которые нужно синхронизировать полностью
	syncResult, err := s.transport.SyncShortPasswords(ctx, shortPasswords)
	if err != nil {
		return err
	}

	s.log.Debug().Msgf("Short sync result: %d local IDs to send, %d server IDs to receive",
		len(syncResult.LocalIDs), len(syncResult.ServerIDs))

	if len(syncResult.LocalIDs) > 0 {
		localPasswords, err := s.storage.GetPasswordsByIDs(syncResult.LocalIDs)
		if err != nil {
			return err
		}

		// 4. отправляем полные данные паролей на сервер
		serverPasswords, err := s.transport.SyncPasswords(ctx, localPasswords)
		if err != nil {
			return err
		}

		// Обрабатываем пароли, возвращенные сервером
		// Отдельно обрабатываем удаленные и обычные пароли
		var passwordsToUpdate []model.Password
		for _, pass := range serverPasswords {
			if pass.Deleted {
				// Сервер помечает пароль как удаленный (был удален на другом устройстве)
				// Помечаем его как удаленный локально
				if pass.LocalID != "" {
					if err := s.storage.DeletePass(pass.LocalID); err != nil {
						s.log.Error().Err(err).Str("local_id", pass.LocalID).Msg("Failed to mark password as deleted")
						continue
					}
					s.log.Info().Str("local_id", pass.LocalID).Msg("Password marked as deleted (deleted on server)")
				}
			} else {
				// Обычное обновление - добавляем в список для обновления
				passwordsToUpdate = append(passwordsToUpdate, pass)
			}
		}

		// Обновляем не удаленные пароли
		if len(passwordsToUpdate) > 0 {
			err = s.storage.UpdatePasswords(&passwordsToUpdate)
			if err != nil {
				return err
			}
		}
	}

	// Получаем пароли с сервера по ServerIDs
	if len(syncResult.ServerIDs) > 0 {
		s.log.Debug().Msgf("Fetching %d passwords from server", len(syncResult.ServerIDs))

		// Запрашиваем все пароли с сервера (включая те, что уже есть локально, чтобы узнать об удалении)
		serverPasswords, err := s.transport.GetPasswordsByServerIDs(ctx, syncResult.ServerIDs)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to fetch passwords from server")
			return err
		}

		var savedCount, updatedCount, deletedCount int
		for _, pass := range serverPasswords {
			if pass.Deleted {
				// Пароль удален на сервере - помечаем его как удаленный локально
				// Ищем локальный пароль по ServerID
				localPasswords, err := s.storage.GetPasswordsByServerIDs([]string{pass.ServerID})
				if err == nil && len(localPasswords) > 0 {
					// Помечаем как удаленный
					if err := s.storage.DeletePass(localPasswords[0].LocalID); err != nil {
						s.log.Error().Err(err).Str("server_id", pass.ServerID).Msg("Failed to mark password as deleted")
						continue
					}
					deletedCount++
					s.log.Info().Str("server_id", pass.ServerID).Msg("Password marked as deleted (deleted on server)")
				}
			} else {
				// Проверяем, существует ли пароль локально по ServerID
				localPasswords, err := s.storage.GetPasswordsByServerIDs([]string{pass.ServerID})
				if err == nil && len(localPasswords) > 0 {
					// Пароль уже существует - обновляем его
					pass.LocalID = localPasswords[0].LocalID
					if err := s.storage.UpdatePass(&pass); err != nil {
						s.log.Error().Err(err).Str("server_id", pass.ServerID).Msg("Failed to update password")
						continue
					}
					updatedCount++
				} else {
					// Пароль не существует - создаем новый
					if err := s.storage.SavePass(&pass); err != nil {
						s.log.Error().Err(err).Str("server_id", pass.ServerID).Msg("Failed to save password")
						continue
					}
					savedCount++
				}
			}
		}

		s.log.Info().Msgf("Successfully processed passwords: %d saved, %d updated, %d marked as deleted", savedCount, updatedCount, deletedCount)
	}

	s.log.Info().Msg("Passwords sync finished.")
	return nil
}

func (s *SyncService) syncFiles(ctx context.Context) error {
	s.log.Info().Msg("Syncing file metadata...")

	// 1. Получаем краткую информацию о файлах
	shortFiles, err := s.storage.GetShortFiles()
	if err != nil {
		return err
	}

	// 2. Отправляем краткую информацию на сервер и получаем ID файлов, которые нужно синхронизировать полностью
	syncResult, err := s.transport.SyncShortFiles(ctx, shortFiles)
	if err != nil {
		return err
	}

	s.log.Debug().Msgf("Short sync result: %d local IDs to send, %d server IDs to receive",
		len(syncResult.LocalIDs), len(syncResult.ServerIDs))

	// 3. Получаем полные метаданные файлов по LocalID (те, которые нужно отправить на сервер)
	var localFiles []model.FileData
	var activeLocalFiles []model.FileData

	if len(syncResult.LocalIDs) > 0 {
		localFiles, err = s.storage.GetFilesByIDs(syncResult.LocalIDs)
		if err != nil {
			return err
		}

		// Фильтруем удаленные файлы перед отправкой на сервер
		activeLocalFiles = make([]model.FileData, 0, len(localFiles))
		for _, file := range localFiles {
			if file.Deleted {
				continue
			}
			file.SyncTime = s.lastSyncTime
			activeLocalFiles = append(activeLocalFiles, file)
		}
		s.log.Debug().Msgf("Prepared %d active files to send to server", len(activeLocalFiles))
	}

	// 4. ВСЕГДА отправляем полные метаданные файлов на сервер (даже если список пустой)
	// Сервер всегда возвращает все данные, которые нужно получить клиенту
	serverFiles, err := s.transport.SyncFiles(ctx, activeLocalFiles)
	if err != nil {
		return err
	}

	// 5. Обрабатываем ответ сервера
	localSyncables := make([]domain.Syncable, len(localFiles))
	for i := range localFiles {
		localSyncables[i] = &localFiles[i]
	}
	serverSyncables := make([]domain.Syncable, len(serverFiles))
	for i := range serverFiles {
		serverSyncables[i] = &serverFiles[i]
	}

	// saveFileFunc := func(data map[string]interface{}) error {
	// 	file, err := model.FromMapFile(data)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to convert map to file model on save: %w", err)
	// 	}
	// 	// Если в данных уже есть LocalID, используем UpdateFile вместо SaveFileMetadata
	// 	if file.LocalID != "" {
	// 		s.log.Warn().Str("local_id", file.LocalID).Msg("File has LocalID in saveFunc, using UpdateFile instead")
	// 		return s.storage.UpdateFile(&file)
	// 	}
	// 	return s.storage.SaveFileMetadata(&file)
	// }
	// updateFileFunc := func(data map[string]interface{}) error {
	// 	file, err := model.FromMapFile(data)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to convert map to file model on update: %w", err)
	// 	}
	// 	return s.storage.UpdateFile(&file)
	// }
	//s.processSyncResults(localSyncables, serverSyncables, saveFileFunc, updateFileFunc, s.storage.DeleteFileByID, "file")

	s.log.Info().Msg("File metadata sync finished.")
	return nil
}
