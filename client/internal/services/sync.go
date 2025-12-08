package services

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/transport"
	"time"

	model "gophKeeper/pkg/grpchelper"

	"github.com/rs/zerolog"
)

// SyncService отвечает за двустороннюю синхронизацию данных с сервером.
type SyncService struct {
	storage      domain.LocalStorage
	log          *zerolog.Logger
	lastSyncTime time.Time // Время последней успешной синхронизации
}

// NewSyncService создает новый экземпляр сервиса синхронизации.
func NewSyncService(storage domain.LocalStorage, log *zerolog.Logger) *SyncService {
	return &SyncService{
		storage: storage,
		log:     log,
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
		s.lastSyncTime = time.Time{}
	}

	// Получаем учетные данные для запросов
	_, token, deviceID, err := s.storage.GetUserCredentials()
	if err != nil {
		return err
	}

	// Синхронизация текстовых заметок
	if err := s.syncTexts(ctx, token, deviceID); err != nil {
		s.log.Error().Err(err).Msg("Text sync failed")
	}

	if err := s.syncCards(ctx, token, deviceID); err != nil {
		s.log.Error().Err(err).Msg("Card sync failed")
	}
	if err := s.syncPasswords(ctx, token, deviceID); err != nil {
		s.log.Error().Err(err).Msg("Password sync failed")
	}
	if err := s.syncFiles(ctx, token, deviceID); err != nil {
		s.log.Error().Err(err).Msg("File metadata sync failed")
	}

	if err := s.storage.SaveLastSyncTime(time.Now()); err != nil {
		s.log.Error().Err(err).Msg("Failed to save last sync time")
		return err
	}

	s.log.Info().Msg("Synchronization completed successfully")
	return nil
}

func (s *SyncService) syncTexts(ctx context.Context, token, deviceID string) error {
	s.log.Info().Msg("Syncing text notes...")

	// 1. Получаем краткую информацию о текстах
	shortTexts, err := s.storage.GetShortTexts()
	if err != nil {
		return err
	}

	// 2. Отправляем краткую информацию на сервер и получаем ID текстов, которые нужно синхронизировать полностью
	textIDsToSync, err := transport.SyncShort(ctx, token, deviceID, shortTexts)
	if err != nil {
		return err
	}

	// 3. Получаем полные данные текстов по ID
	localTexts, err := s.storage.GetTextsByIDs(textIDsToSync)
	if err != nil {
		return err
	}

	// Фильтруем удаленные тексты
	activeLocalTexts := make([]model.TextData, 0, len(localTexts))
	for _, text := range localTexts {
		if text.Deleted {
			continue
		}
		text.SyncTime = s.lastSyncTime
		activeLocalTexts = append(activeLocalTexts, text)
	}

	// 4. Отправляем полные данные текстов на сервер
	serverTexts, err := transport.SyncTexts(ctx, token, deviceID, activeLocalTexts)
	if err != nil {
		return err
	}

	// 5. Обрабатываем ответ сервера
	localSyncables := make([]domain.Syncable, len(localTexts))
	for i := range localTexts {
		localSyncables[i] = &localTexts[i]
	}
	serverSyncables := make([]domain.Syncable, len(serverTexts))
	for i := range serverTexts {
		serverSyncables[i] = &serverTexts[i]
	}

	saveTextFunc := func(data map[string]interface{}) error {
		text, err := model.FromMapText(data)
		if err != nil {
			return fmt.Errorf("failed to convert map to text model on save: %w", err)
		}
		return s.storage.SaveText(&text)
	}
	updateTextFunc := func(data map[string]interface{}) error {
		text, err := model.FromMapText(data)
		if err != nil {
			return fmt.Errorf("failed to convert map to text model on update: %w", err)
		}
		return s.storage.UpdateText(&text)
	}

	s.processSyncResults(localSyncables, serverSyncables, saveTextFunc, updateTextFunc,
		s.storage.DeleteText,
		"text",
	)

	s.log.Info().Msg("Text notes sync finished.")
	return nil
}

func (s *SyncService) syncCards(ctx context.Context, token, deviceID string) error {
	s.log.Info().Msg("Syncing credit cards...")

	// 1. Получаем краткую информацию о картах
	shortCards, err := s.storage.GetShortCards()
	if err != nil {
		return err
	}

	// 2. Отправляем краткую информацию на сервер и получаем ID карт, которые нужно синхронизировать полностью
	cardIDsToSync, err := transport.SyncShort(ctx, token, deviceID, shortCards)
	if err != nil {
		return err
	}

	// 3. Получаем полные данные карт по ID
	localCards, err := s.storage.GetCardsByIDs(cardIDsToSync)
	if err != nil {
		return err
	}

	// Фильтруем удаленные карты
	activeLocalCards := make([]model.Card, 0, len(localCards))
	for _, card := range localCards {
		if card.Deleted {
			continue
		}
		card.SyncTime = s.lastSyncTime
		activeLocalCards = append(activeLocalCards, card)
	}

	// 4. Отправляем полные данные карт на сервер
	serverCards, err := transport.SyncCards(ctx, token, deviceID, activeLocalCards)
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

	saveCardFunc := func(data map[string]interface{}) error {
		card, err := model.FromMapCard(data)
		if err != nil {
			return fmt.Errorf("failed to convert map to card model on save: %w", err)
		}
		return s.storage.SaveCard(&card)
	}
	updateCardFunc := func(data map[string]interface{}) error {
		card, err := model.FromMapCard(data)
		if err != nil {
			return fmt.Errorf("failed to convert map to card model on update: %w", err)
		}
		return s.storage.UpdateCard(&card)
	}

	s.processSyncResults(localSyncables, serverSyncables, saveCardFunc, updateCardFunc,
		s.storage.DeleteCard,
		"card",
	)

	s.log.Info().Msg("Credit cards sync finished.")
	return nil
}

func (s *SyncService) syncPasswords(ctx context.Context, token, deviceID string) error {
	s.log.Info().Msg("Syncing passwords...")

	// 1. Получаем краткую информацию о паролях
	shortPasswords, err := s.storage.GetShortPasswords()
	if err != nil {
		return err
	}

	// 2. Отправляем краткую информацию на сервер и получаем ID паролей, которые нужно синхронизировать полностью
	passwordIDsToSync, err := transport.SyncShort(ctx, token, deviceID, shortPasswords)
	if err != nil {
		return err
	}

	// 3. Получаем полные данные паролей по ID
	localPasswords, err := s.storage.GetPasswordsByIDs(passwordIDsToSync)
	if err != nil {
		return err
	}

	// Фильтруем удаленные пароли
	activeLocalPasswords := make([]model.Password, 0, len(localPasswords))
	for _, pass := range localPasswords {
		if pass.Deleted {
			continue
		}
		pass.SyncTime = s.lastSyncTime
		activeLocalPasswords = append(activeLocalPasswords, pass)
	}

	// 4. Отправляем полные данные паролей на сервер
	serverPasswords, err := transport.SyncPasswords(ctx, token, deviceID, activeLocalPasswords)
	if err != nil {
		return err
	}

	// 5. Обрабатываем ответ сервера
	localSyncables := make([]domain.Syncable, len(localPasswords))
	for i := range localPasswords {
		localSyncables[i] = &localPasswords[i]
	}
	serverSyncables := make([]domain.Syncable, len(serverPasswords))
	for i := range serverPasswords {
		serverSyncables[i] = &serverPasswords[i]
	}

	savePassFunc := func(data map[string]interface{}) error {
		pass, err := model.FromMapPassword(data)
		if err != nil {
			return fmt.Errorf("failed to convert map to password model on save: %w", err)
		}
		return s.storage.SavePass(&pass)
	}
	updatePassFunc := func(data map[string]interface{}) error {
		pass, err := model.FromMapPassword(data)
		if err != nil {
			return fmt.Errorf("failed to convert map to password model on update: %w", err)
		}
		return s.storage.UpdatePass(&pass)
	}

	s.processSyncResults(localSyncables, serverSyncables, savePassFunc, updatePassFunc,
		s.storage.DeletePass,
		"password",
	)

	s.log.Info().Msg("Passwords sync finished.")
	return nil
}

func (s *SyncService) syncFiles(ctx context.Context, token, deviceID string) error {
	s.log.Info().Msg("Syncing file metadata...")

	// 1. Получаем краткую информацию о файлах
	shortFiles, err := s.storage.GetShortFiles()
	if err != nil {
		return err
	}

	// 2. Отправляем краткую информацию на сервер и получаем ID файлов, которые нужно синхронизировать полностью
	fileIDsToSync, err := transport.SyncShort(ctx, token, deviceID, shortFiles)
	if err != nil {
		return err
	}

	// 3. Получаем полные метаданные файлов по ID
	localFiles, err := s.storage.GetFilesByIDs(fileIDsToSync)
	if err != nil {
		return err
	}

	// Фильтруем удаленные файлы
	activeLocalFiles := make([]model.FileData, 0, len(localFiles))
	for _, file := range localFiles {
		if file.Deleted {
			continue
		}
		file.SyncTime = s.lastSyncTime
		activeLocalFiles = append(activeLocalFiles, file)
	}

	// 4. Отправляем полные метаданные файлов на сервер
	serverFiles, err := transport.SyncFiles(ctx, token, deviceID, activeLocalFiles)
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

	saveFileFunc := func(data map[string]interface{}) error {
		file, err := model.FromMapFile(data)
		if err != nil {
			return fmt.Errorf("failed to convert map to file model on save: %w", err)
		}
		return s.storage.SaveFileMetadata(&file)
	}
	updateFileFunc := func(data map[string]interface{}) error {
		file, err := model.FromMapFile(data)
		if err != nil {
			return fmt.Errorf("failed to convert map to file model on update: %w", err)
		}
		return s.storage.UpdateFile(&file)
	}
	s.processSyncResults(localSyncables, serverSyncables, saveFileFunc, updateFileFunc, s.storage.DeleteFileByID, "file")

	s.log.Info().Msg("File metadata sync finished.")
	return nil
}

// processSyncResults — это универсальный метод для обработки результатов синхронизации.
func (s *SyncService) processSyncResults(
	localItems []domain.Syncable,
	serverItems []domain.Syncable,
	saveFunc func(map[string]interface{}) error,
	updateFunc func(map[string]interface{}) error,
	deleteFunc func(string) error,
	entityName string,
) {
	// Создаем map для быстрого доступа к локальным элементам по их LocalID.
	localItemsMap := make(map[string]domain.Syncable)
	for _, item := range localItems {
		localItemsMap[item.GetLocalID()] = item
	}
	serverItemsSet := make(map[string]bool)

	for _, serverItem := range serverItems {
		serverLocalID := serverItem.GetLocalID()
		serverItemsSet[serverLocalID] = true // Отмечаем, что этот элемент пришел с сервера
		localItem, found := localItemsMap[serverLocalID]

		// Проверяем, не помечен ли элемент как удаленный на сервере
		if itemWithDelete, ok := serverItem.(interface{ GetDeleted() bool }); ok && itemWithDelete.GetDeleted() {
			if found { // Если элемент еще существует локально, помечаем его как удаленный
				s.log.Info().Str("local_id", serverLocalID).Msgf("Marking local %s as deleted, following server state", entityName)
				// Мы не удаляем запись, а обновляем ее, устанавливая флаг Deleted
				dataToSave := serverItem.ToMap()
				updateFunc(dataToSave)
			}
			continue // Переходим к следующему элементу
		}

		dataToSave := serverItem.ToMap()

		if !found {
			s.log.Info().Str("local_id", serverLocalID).Msgf("Creating new local %s from server", entityName)
			if err := saveFunc(dataToSave); err != nil {
				s.log.Error().Err(err).Str("local_id", serverLocalID).Msgf("Failed to save new local %s", entityName)
			}
			continue
		}

		if localItem.GetServerID() == "" && serverItem.GetServerID() != "" {
			s.log.Info().Str("local_id", serverLocalID).Str("server_id", serverItem.GetServerID()).Msgf("Local %s synced, saving server_id", entityName)
			if err := updateFunc(dataToSave); err != nil {
				s.log.Error().Err(err).Str("local_id", serverLocalID).Msgf("Failed to update %s with server_id", entityName)
			}
		} else if serverItem.GetChangeTime().After(localItem.GetChangeTime()) {
			s.log.Info().Str("local_id", serverLocalID).Msgf("Updating local %s from server (newer version found)", entityName)
			if err := updateFunc(dataToSave); err != nil {
				s.log.Error().Err(err).Str("local_id", serverLocalID).Msgf("Failed to update local %s from server", entityName)
			}
		}
	}

}
