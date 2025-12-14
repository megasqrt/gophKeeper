package storage

import (
	"crypto/sha256"
	"encoding/hex"
	model "gophKeeper/pkg/grpchelper"

)

// calculateTextChecksum вычисляет checksum для текстовой записи на основе всех полей данных.
func calculateTextChecksum(text *model.TextData) string {
	h := sha256.New()
	h.Write([]byte(text.Title))
	h.Write([]byte(text.Text))
	return hex.EncodeToString(h.Sum(nil))
}
