package storage

import (
	"crypto/sha256"
	"encoding/hex"
	model "gophKeeper/pkg/grpchelper"
)

// calculateCardChecksum вычисляет checksum для карты на основе всех полей данных.
func calculateCardChecksum(card *model.Card) string {
	h := sha256.New()
	h.Write([]byte(card.Number))
	h.Write([]byte(card.Holder))
	h.Write([]byte(card.Expiry))
	h.Write([]byte(card.CVV))
	return hex.EncodeToString(h.Sum(nil))
}
