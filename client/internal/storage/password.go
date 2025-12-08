package storage

import (
	"gophKeeper/client/internal/domain/model"
	"time"
)

// SavePass сохраняет данные пароля в хранилище.
func (s *BboltStorage) SavePass(passData *model.Password) error {
	s.log.Info().Msg("Saving new password")
	passData.ChangeTime = time.Now()
	return s.saveItem(passwordsBucket, passData, true)
}

// UpdatePass обновляет данные существующего пароля.
func (s *BboltStorage) UpdatePass(passData *model.Password) error {
	s.log.Info().Str("pass_id", passData.LocalID).Msg("Updating password")
	passData.ChangeTime = time.Now()
	return s.saveItem(passwordsBucket, passData, false)
}

// GetPasss извлекает все сохраненные пароли.
func (s *BboltStorage) GetPasss() ([]model.Password, error) {
	s.log.Info().Msg("Retrieving all passwords from storage")
	items, err := s.getAllItems(passwordsBucket, func(data map[string]interface{}) (interface{}, error) {
		return model.FromMapPassword(data)
	})
	if err != nil {
		return nil, err
	}

	var passwords []model.Password
	for _, item := range items {
		passwords = append(passwords, item.(model.Password))
	}
	return passwords, nil
}

// DeletePass помечает пароль как удаленный.
func (s *BboltStorage) DeletePass(id string) error {
	return s.markAsDeleted(passwordsBucket, id, func(data map[string]interface{}) (interface{}, error) {
		return model.FromMapPassword(data)
	})
}
