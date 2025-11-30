package storage

import "time"

// SavePass сохраняет данные пароля в хранилище.
func (s *BboltStorage) SavePass(passData map[string]string) error {
	s.log.Info().Msg("Saving new password")
	passData["changeTime"] = time.Now().Format(time.RFC3339Nano)
	return s.saveItem(passwordsBucket, StringMapToInterfaceMap(passData), true)
}

// UpdatePass обновляет данные существующего пароля.
func (s *BboltStorage) UpdatePass(passData map[string]string) error {
	s.log.Info().Str("pass_id", passData["id"]).Msg("Updating password")
	passData["changeTime"] = time.Now().Format(time.RFC3339Nano)
	return s.saveItem(passwordsBucket, StringMapToInterfaceMap(passData), false)
}

// GetPasss извлекает все сохраненные пароли.
func (s *BboltStorage) GetPasss() ([]map[string]string, error) {
	s.log.Info().Msg("Retrieving all passwords from storage")
	items, err := s.getAllItems(passwordsBucket)
	if err != nil {
		return nil, err
	}

	var passwords []map[string]string
	for _, item := range items {
		passwords = append(passwords, InterfaceMapToStringMap(item))
	}
	return passwords, nil
}
