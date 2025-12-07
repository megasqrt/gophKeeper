package services

import (
	"context"
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/domain/model"
	"gophKeeper/client/internal/transport"
	"time"

	"github.com/rs/zerolog"
)

// SyncService отвечает за двустороннюю синхронизацию данных с сервером.
type SyncService struct {
	storage domain.LocalStorage
	log     *zerolog.Logger
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

	lastSyncTime, err := s.storage.GetLastSyncTime()
	if err != nil {
		s.log.Warn().Err(err).Msg("Could not get last sync time, performing full sync")
		// Если времени нет, используем нулевое время, чтобы синхронизировать все.
		lastSyncTime = time.Time{}
	}

	// Получаем учетные данные для запросов
	_, token, deviceID, err := s.storage.GetUserCredentials()
	if err != nil {
		return err
	}

	// Синхронизация текстовых заметок
	if err := s.syncTexts(ctx, token, deviceID, lastSyncTime); err != nil {
		s.log.Error().Err(err).Msg("Text sync failed")
		// Можно либо прервать всю синхронизацию, либо продолжить с другими типами данных.
		// Пока что продолжим.
	}

	if err := s.syncCards(ctx, token, deviceID, lastSyncTime); err != nil {
		s.log.Error().Err(err).Msg("Card sync failed")
	}
	if err := s.syncPasswords(ctx, token, deviceID, lastSyncTime); err != nil { 
		s.log.Error().Err(err).Msg("Password sync failed")
	}
	if err := s.syncFiles(ctx, token, deviceID, lastSyncTime); err != nil {
		s.log.Error().Err(err).Msg("File metadata sync failed")
	}

	// Если все прошло успешно, обновляем время последней синхронизации.
	if err := s.storage.SaveLastSyncTime(time.Now()); err != nil {
		s.log.Error().Err(err).Msg("Failed to save last sync time")
		return err
	}

	s.log.Info().Msg("Synchronization completed successfully")
	return nil
}

func (s *SyncService) syncTexts(ctx context.Context, token, deviceID string, lastSync time.Time) error {
	s.log.Info().Msg("Syncing text notes...")

	// 1. Получаем все локальные тексты
	localTextsData, err := s.storage.GetTexts()
	if err != nil {
		return err
	}
	// Конвертируем данные из хранилища в доменную модель
	localTexts := make([]model.TextData, len(localTextsData))
	for i, data := range localTextsData {
		textData := model.TextData{
			ID:    data["id"],
			Title: data["title"],
			Text:  data["text"],
		}
		if changeTimeStr, ok := data["changeTime"]; ok {
			textData.ChangeTime, _ = time.Parse(time.RFC3339Nano, changeTimeStr)
		}
		localTexts[i] = textData
	}

	// 2. Отправляем на сервер
	serverTexts, err := transport.SyncTexts(ctx, token, deviceID, localTexts)
	if err != nil {
		return err
	}

	// 3. Обрабатываем ответ сервера
	for _, serverText := range serverTexts {
		// Ищем соответствующую локальную запись
		var foundLocal *model.TextData
		for i := range localTexts {
			if localTexts[i].ID == serverText.ID {
				foundLocal = &localTexts[i]
				break
			}
		}

		if foundLocal == nil {
			// Записи нет локально, создаем ее
			s.log.Info().Str("id", serverText.ID).Str("title", serverText.Title).Msg("Creating new local text from server")
			textMap := map[string]string{
				"id":         serverText.ID,
				"title":      serverText.Title,
				"text":       serverText.Text,
				"changeTime": serverText.ChangeTime.Format(time.RFC3339Nano),
			}
			if err := s.storage.SaveText(textMap); err != nil {
				s.log.Error().Err(err).Str("id", serverText.ID).Msg("Failed to save new local text")
			}
		} else {
			// Запись есть, сравниваем время изменения
			if serverText.ChangeTime.After(foundLocal.ChangeTime) {
				// На сервере новее, обновляем локальную
				textMap := map[string]string{
					"id":         serverText.ID,
					"title":      serverText.Title,
					"text":       serverText.Text,
					"changeTime": serverText.ChangeTime.Format(time.RFC3339Nano),
				}
				s.log.Info().Str("id", serverText.ID).Str("title", serverText.Title).Msg("Updating local text from server")
				if err := s.storage.UpdateText(textMap); err != nil {
					s.log.Error().Err(err).Str("id", serverText.ID).Msg("Failed to update local text")
				}
			}
			// Если локальная новее, мы уже отправили ее на шаге 2, и сервер должен был ее обновить.
			// Ничего делать не нужно.
		}
	}

	// 4. Проверяем, не были ли какие-то записи удалены на сервере
	// (Сервер мог бы вернуть список ID, которые нужно удалить)
	// Это более сложная логика, пока пропустим.

	s.log.Info().Msg("Text notes sync finished.")
	return nil
}

func (s *SyncService) syncCards(ctx context.Context, token, deviceID string, lastSync time.Time) error {
	s.log.Info().Msg("Syncing credit cards...")

	// 1. Получаем все локальные карты
	localCardsData, err := s.storage.GetCards()
	if err != nil {
		return err
	}

	localCards := make([]model.Card, len(localCardsData))
	for i, data := range localCardsData {
		card := model.Card{
			ID:     data["id"],
			Number: data["number"],
			Holder: data["holder"],
			Expiry: data["expiry"],
			CVV:    data["cvv"],
		}
		if changeTimeStr, ok := data["changeTime"]; ok {
			card.ChangeTime, _ = time.Parse(time.RFC3339Nano, changeTimeStr)
		}
		localCards[i] = card
	}

	// 2. Отправляем на сервер (этот метод нужно будет создать в transport и grpc клиенте)
	serverCards, err := transport.SyncCards(ctx, token, deviceID, localCards)
	if err != nil {
		return err
	}

	// 3. Обрабатываем ответ сервера
	for _, serverCard := range serverCards {
		var foundLocal *model.Card
		for i := range localCards {
			if localCards[i].ID == serverCard.ID {
				foundLocal = &localCards[i]
				break
			}
		}

		cardMap := map[string]string{
			"id":         serverCard.ID,
			"number":     serverCard.Number,
			"holder":     serverCard.Holder,
			"expiry":     serverCard.Expiry,
			"cvv":        serverCard.CVV,
			"changeTime": serverCard.ChangeTime.Format(time.RFC3339Nano),
		}

		if foundLocal == nil {
			s.log.Info().Str("id", serverCard.ID).Msg("Creating new local card from server")
			s.storage.SaveCard(cardMap)
		} else if serverCard.ChangeTime.After(foundLocal.ChangeTime) {
			s.log.Info().Str("id", serverCard.ID).Msg("Updating local card from server")
			s.storage.UpdateCard(cardMap)
		}
	}

	s.log.Info().Msg("Credit cards sync finished.")
	return nil
}

func (s *SyncService) syncPasswords(ctx context.Context, token, deviceID string, lastSync time.Time) error {
	s.log.Info().Msg("Syncing passwords...")

	// 1. Получаем все локальные пароли
	localPassData, err := s.storage.GetPasss()
	if err != nil {
		return err
	}

	localPass := make([]model.Password, len(localPassData))
	for i, data := range localPassData {
		pass := model.Password{
			ID:          data["id"],
			Login:       data["login"],
			Password:    data["password"],
			Description: data["description"],
		}
		if changeTimeStr, ok := data["changeTime"]; ok {
			pass.ChangeTime, _ = time.Parse(time.RFC3339Nano, changeTimeStr)
		}
		localPass[i] = pass
	}

	// 2. Отправляем на сервер
	serverPass, err := transport.SyncPasswords(ctx, token, deviceID, localPass)
	if err != nil {
		return err
	}

	// 3. Обрабатываем ответ сервера
	for _, sp := range serverPass {
		var foundLocal *model.Password
		for i := range localPass {
			if localPass[i].ID == sp.ID {
				foundLocal = &localPass[i]
				break
			}
		}

		passMap := map[string]string{
			"id":          sp.ID,
			"login":       sp.Login,
			"password":    sp.Password,
			"description": sp.Description,
			"changeTime":  sp.ChangeTime.Format(time.RFC3339Nano),
		}

		if foundLocal == nil {
			s.log.Info().Str("id", sp.ID).Msg("Creating new local password from server")
			if err := s.storage.SavePass(passMap); err != nil {
				s.log.Error().Err(err).Str("id", sp.ID).Msg("Failed to save new local password")
			}
		} else if sp.ChangeTime.After(foundLocal.ChangeTime) {
			s.log.Info().Str("id", sp.ID).Msg("Updating local password from server")
			if err := s.storage.UpdatePass(passMap); err != nil {
				s.log.Error().Err(err).Str("id", sp.ID).Msg("Failed to update local password")
			}
		}
	}

	s.log.Info().Msg("Passwords sync finished.")
	return nil
}

func (s *SyncService) syncFiles(ctx context.Context, token, deviceID string, lastSync time.Time) error {
	s.log.Info().Msg("Syncing file metadata...")

	// 1. Получаем все локальные метаданные файлов
	localFilesData, err := s.storage.GetFiles()
	if err != nil {
		return err
	}

	localFiles := make([]model.FileData, len(localFilesData))
	for i, data := range localFilesData {
		file := model.FileData{
			ID:       data["id"].(string),
			Name:     data["name"].(string),
			Metadata: data["metadata"].(string),
		}
		// JSON unmarshal в map[string]interface{} преобразует числа в float64
		if size, ok := data["size"].(float64); ok {
			file.Size = int64(size)
		}
		if changeTimeStr, ok := data["changeTime"].(string); ok {
			file.ChangeTime, _ = time.Parse(time.RFC3339Nano, changeTimeStr)
		}
		localFiles[i] = file
	}

	// 2. Отправляем на сервер
	serverFiles, err := transport.SyncFiles(ctx, token, deviceID, localFiles)
	if err != nil {
		return err
	}

	// 3. Обрабатываем ответ сервера
	for _, sf := range serverFiles {
		var foundLocal *model.FileData
		for i := range localFiles {
			if localFiles[i].ID == sf.ID {
				foundLocal = &localFiles[i]
				break
			}
		}

		// Для файлов мы сохраняем данные как map[string]interface{}
		fileMap := map[string]interface{}{
			"id":         sf.ID,
			"name":       sf.Name,
			"metadata":   sf.Metadata,
			"size":       sf.Size,
			"changeTime": sf.ChangeTime.Format(time.RFC3339Nano),
		}

		if foundLocal == nil {
			s.log.Info().Str("id", sf.ID).Str("name", sf.Name).Msg("Creating new local file metadata from server")
			// При создании новой записи о файле, самого файла у нас еще нет.
			// Мы просто сохраняем метаданные.
			if err := s.storage.SaveFile(fileMap); err != nil {
				s.log.Error().Err(err).Str("id", sf.ID).Msg("Failed to save new local file metadata")
			}
		} else if sf.ChangeTime.After(foundLocal.ChangeTime) {
			s.log.Info().Str("id", sf.ID).Str("name", sf.Name).Msg("Updating local file metadata from server")
			// TODO: Здесь должна быть логика для пометки файла как "требующий скачивания"
			// Пока просто обновляем метаданные.
		}
	}

	s.log.Info().Msg("File metadata sync finished.")
	return nil
}
