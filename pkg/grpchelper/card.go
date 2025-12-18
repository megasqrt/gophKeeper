package grpchelper

import (
	"fmt"
	"strings"
)

// Типы для хранения token и deviceID в контексте (должны совпадать с services пакетом)
type ContextKey string

const (
	TokenKey    ContextKey = "token"
	DeviceIDKey ContextKey = "deviceID"
	UserKey     ContextKey = ""
)

// Card представляет собой данные кредитной карты.
type Card struct {
	LocalID    int64
	ServerID   int64
	Number     string
	Holder     string
	Expiry     string
	CVV        string
	Metadata   string
	ChangeTime int64
	Deleted    bool
	Checksum   string
	Version	   int32
}

func (c Card) GetLocalID() int64       { return c.LocalID }
func (c Card) GetServerID() int64      { return c.ServerID }
func (c Card) GetChangeTime() int64     { return c.ChangeTime }
func (c *Card) SetLocalID(id int64)    { c.LocalID = id }
func (c Card) GetDeleted() bool         { return c.Deleted }
func (c *Card) SetDeleted(deleted bool) { c.Deleted = deleted }
func (c Card) GetChecksum() string      { return c.Checksum }

// Title возвращает заголовок для элемента списка (номер карты).
func (c Card) Title() string {
	// Маскируем номер карты для безопасности
	if len(c.Number) > 4 {
		return fmt.Sprintf("**** **** **** %s", c.Number[len(c.Number)-4:])
	}
	return "****"
}

// Description возвращает описание для элемента списка (держатель карты).
func (c Card) Description() string { return strings.ToUpper(c.Holder) }

// FilterValue используется для фильтрации списка.
func (c Card) FilterValue() string { return c.Title() + " " + c.Description() }
