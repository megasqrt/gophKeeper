package services

import (
	"context"
	"gophKeeper/client/internal/transport"
	model "gophKeeper/pkg/grpchelper"
)

// TransportInterface определяет интерфейс для transport слоя
// Это позволяет легко мокировать transport в тестах
type TransportInterface interface {
	SyncTexts(ctx context.Context, localTexts []model.TextData) ([]model.TextData, error)
	SyncCards(ctx context.Context, localCards []model.Card) ([]model.Card, error)
	SyncPasswords(ctx context.Context, localPasswords []model.Password) ([]model.Password, error)
	SyncFiles(ctx context.Context, localFiles []model.FileData) ([]model.FileData, error)
	GetPasswordsByServerIDs(ctx context.Context, serverPasswordIds []string) ([]model.Password, error)
}

// defaultTransport реализует TransportInterface используя реальный transport пакет
type defaultTransport struct{}


func (d *defaultTransport) SyncTexts(ctx context.Context, localTexts []model.TextData) ([]model.TextData, error) {
	return transport.SyncTexts(ctx, localTexts)
}


func (d *defaultTransport) SyncCards(ctx context.Context, localCards []model.Card) ([]model.Card, error) {
	return transport.SyncCards(ctx, localCards)
}


func (d *defaultTransport) SyncPasswords(ctx context.Context, localPasswords []model.Password) ([]model.Password, error) {
	return transport.SyncPasswords(ctx, localPasswords)
}

func (d *defaultTransport) GetPasswordsByServerIDs(ctx context.Context, serverPasswordIds []string) ([]model.Password, error) {
	return transport.GetPasswordsByServerIDs(ctx, serverPasswordIds)
}

func (d *defaultTransport) SyncFiles(ctx context.Context, localFiles []model.FileData) ([]model.FileData, error) {
	return transport.SyncFiles(ctx, localFiles)
}
