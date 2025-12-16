package services

import (
	"context"
	pb "gophKeeper/pkg/proto"
	"gophKeeper/server/internal/domain/repository"

	"github.com/rs/zerolog"

)

// CardService реализует gRPC сервис для работы с картами.
type CardService struct {
	pb.UnimplementedCardServiceServer
	log      zerolog.Logger
	cardRepo repository.CardRepository
}

// NewCardService создает новый экземпляр сервиса для карт.
func NewCardService(log zerolog.Logger, cardRepo repository.CardRepository) *CardService {
	return &CardService{
		log:      log,
		cardRepo: cardRepo,
	}
}

// CardsSync выполняет полную синхронизацию карт.
func (s *CardService) CardsSync(ctx context.Context, req *pb.CardsSyncRequest) (*pb.GetCardsResponse, error) {
	

	var cardsToClient []*pb.CardItem

	

	s.log.Info().Msgf("Sending %d cards back to client", len(cardsToClient))
	return pb.GetCardsResponse_builder{Cards: cardsToClient}.Build(), nil
}
