package storage

import (
	"crypto/sha256"
	"encoding/hex"
	model "gophKeeper/pkg/grpchelper"
	"time"

	"go.etcd.io/bbolt"
)

// calculatePasswordChecksum вычисляет checksum для пароля на основе всех полей данных.
func calculatePasswordChecksum(pass *model.Password) string {
	h := sha256.New()
	h.Write([]byte(pass.Login))
	h.Write([]byte(pass.Password))
	h.Write([]byte(pass.Description))
	return hex.EncodeToString(h.Sum(nil))
}

// SavePass сохраняет данные пароля в хранилище.
func (s *BboltStorage) SavePass(passData *model.Password) error {
	s.log.Info().Msg("Saving new password")
	passData.ChangeTime = time.Now().Unix()
	passData.Checksum = calculatePasswordChecksum(passData)
	return s.saveItem(passwordsBucket, passData, true)
}

// UpdatePass обновляет данные существующего пароля.
func (s *BboltStorage) UpdatePass(passData *model.Password) error {
	s.log.Info().Str("pass_id", passData.LocalID).Msg("Updating password")
	passData.ChangeTime = time.Now().Unix()
	passData.Checksum = calculatePasswordChecksum(passData)
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

// GetShortPasswords извлекает краткую информацию о паролях для синхронизации.
func (s *BboltStorage) GetShortPasswords() ([]model.SyncInfo, error) {
	s.log.Info().Msg("Retrieving all info passwords from storage")
	var syncInfos []model.SyncInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(passwordsBucket)
		return b.ForEach(func(k, v []byte) error {
			var pass model.Password
			if err := s.decryptItem(v, &pass); err != nil {
				s.log.Error().Err(err).Bytes("key", k).Msg("Could not decrypt password for sync info")
				return nil // Пропускаем поврежденные записи
			}

			syncInfos = append(syncInfos, model.SyncInfo{
				LocalID:  pass.LocalID,
				ServerID: pass.ServerID,
				Checksum: pass.Checksum,
				Deleted:  pass.Deleted,
			})
			return nil
		})
	})

	return syncInfos, err
}

// GetPasswordsByIDs извлекает пароли по их идентификаторам.
func (s *BboltStorage) GetPasswordsByIDs(ids []string) ([]model.Password, error) {
	s.log.Info().Int("count", len(ids)).Msg("Retrieving passwords by IDs from storage")
	var passwords []model.Password

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(passwordsBucket)
		for _, id := range ids {
			encryptedData := b.Get([]byte(id))
			if encryptedData == nil {
				s.log.Warn().Str("pass_id", id).Msg("Password not found, skipping")
				continue
			}

			var pass model.Password
			if err := s.decryptItem(encryptedData, &pass); err != nil {
				s.log.Error().Str("pass_id", id).Err(err).Msg("Failed to decrypt password")
				continue
			}

			passwords = append(passwords, pass)
		}
		return nil
	})

	return passwords, err
}

// DeletePass помечает пароль как удаленный.
func (s *BboltStorage) DeletePass(id string) error {
	return s.markAsDeleted(passwordsBucket, id, func() interface{} {
		return &model.Password{}
	})
}

// DeleteHardPass физически удаляет пароль из хранилища.
func (s *BboltStorage) DeleteHardPass(id string) error {
	return s.deleteItem(passwordsBucket, id)
}
