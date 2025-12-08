package repository

import (
	"context"
	model "gophKeeper/pkg/grpchelper"

	"github.com/google/uuid"
)

type NoteRepository interface {
	Create(ctx context.Context, note *model.TextData) error
	Update(ctx context.Context, note *model.TextData) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.TextData, error)
}
