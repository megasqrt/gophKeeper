package storage

import (
	model "gophKeeper/pkg/grpchelper"
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
	items, err := s.getAllItems(passwordsBucket, func() interface{} {
		return &model.Password{}
	})
	if err != nil {
		return nil, err
	}

	var passwords []model.Password
	for _, item := range items {
		if pass, ok := item.(*model.Password); ok {
			passwords = append(passwords, *pass)
		}
	}
	return passwords, nil
}

// DeletePass помечает пароль как удаленный.
func (s *BboltStorage) DeletePass(id string) error {
	return s.markAsDeleted(passwordsBucket, id, func() interface{} {
		return &model.Password{}
	})
}
