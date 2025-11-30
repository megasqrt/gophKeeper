package model

import (
	"fmt"
	"strings"
	"time"
)

// Card представляет собой данные кредитной карты.
type Card struct {
	ID         string
	Number     string
	Holder     string
	Expiry     string
	CVV        string
	ChangeTime time.Time
	SyncTime   time.Time
}

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

