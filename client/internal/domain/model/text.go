package model

import "time"

type TextData struct {
	ID         string
	Title      string
	Text       string
	ChangeTime time.Time
	SyncTime   time.Time
}

func (i TextData) getShortTitle() string { 
    title := i.Title
    if title == "" {
        return "Untitled"
    }
    // Обрезаем длинные заголовки
    if len(title) > 30 {
        return title[:27] + "..."
    }
    return title
}

func (i TextData) FilterValue() string { return i.Title }