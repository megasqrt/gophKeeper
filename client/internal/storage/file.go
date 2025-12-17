package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	model "gophKeeper/pkg/grpchelper"

)

// calculateFileChecksum вычисляет checksum для файла на основе метаданных (Name, Size, Metadata).
// Содержимое файла не включается в checksum для экономии ресурсов.
func calculateFileChecksum(file *model.FileData) string {
	h := sha256.New()
	h.Write([]byte(file.Name))
	h.Write([]byte(fmt.Sprintf("%d", file.Size)))
	h.Write([]byte(file.Metadata))
	return hex.EncodeToString(h.Sum(nil))
}