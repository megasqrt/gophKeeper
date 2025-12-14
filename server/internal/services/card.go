package services

import (
	"context"
	clientModel "gophKeeper/pkg/grpchelper"
	pb "gophKeeper/pkg/proto"
	"gophKeeper/server/internal/domain/model"
	"gophKeeper/server/internal/domain/repository"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	// Извлекаем userID из контекста (из JWT токена)
	userID, ok := ctx.Value("userID").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user credentials")
	}

	s.log.Info().Msgf("Received %d cards to sync", len(req.GetCards()))

	// Конвертируем protobuf карты в клиентские модели
	clientCards := make(map[string]*clientModel.Card)
	for _, pbCard := range req.GetCards() {
		card := clientModel.FromProtoCard(pbCard)
		// Используем LocalID клиента как ключ
		clientCards[card.LocalID] = &card
	}

	// Получаем все карты пользователя из БД
	serverCards, err := s.cardRepo.GetByUserID(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get cards from db")
		return nil, status.Error(codes.Internal, "failed to retrieve server data")
	}

	// Создаем карту серверных карт по ID
	serverCardsMap := make(map[uuid.UUID]*model.Card)
	for _, card := range serverCards {
		serverCardsMap[card.ID] = card
	}

	var cardsToClient []*pb.CardItem

	// Обрабатываем карты от клиента
	for localID, clientCard := range clientCards {
		// Если карта удалена на клиенте, помечаем ее как удаленную на сервере
		if clientCard.Deleted {
			if clientCard.ServerID != "" {
				serverID, err := uuid.Parse(clientCard.ServerID)
				if err == nil {
					if err := s.cardRepo.Delete(ctx, serverID, userID); err != nil {
						s.log.Error().Err(err).Str("server_id", serverID.String()).Msg("failed to delete card")
					} else {
						s.log.Info().Str("server_id", serverID.String()).Msg("Card marked as deleted by client")
					}
				}
			}
			continue
		}

		if clientCard.ServerID == "" {
			// Новая карта от клиента
			dbCard := &model.Card{
				ID:        uuid.New(),
				UserID:    userID,
				Number:    clientCard.Number,
				Holder:    clientCard.Holder,
				Expiry:    clientCard.Expiry,
				CVV:       clientCard.CVV,
				Metadata:  clientCard.Metadata,
				Checksum:  clientCard.Checksum,
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			}

			if err := s.cardRepo.Create(ctx, dbCard); err != nil {
				s.log.Error().Err(err).Msgf("failed to create card for local_id %s", localID)
				continue
			}

			// Конвертируем обратно в клиентскую модель для отправки
			// responseCard := clientModel.Card{
			// 	LocalID:    localID,
			// 	ServerID:   dbCard.ID.String(),
			// 	Number:     dbCard.Number,
			// 	Holder:     dbCard.Holder,
			// 	Expiry:     dbCard.Expiry,
			// 	CVV:        dbCard.CVV,
			// 	Metadata:   dbCard.Metadata,
			// 	Checksum:   dbCard.Checksum,
			// 	ChangeTime: dbCard.UpdatedAt,
			// 	SyncTime:   dbCard.UpdatedAt,
			// 	Deleted:    false,
			// }
			//cardsToClient = append(cardsToClient, responseCard.ToProto())
		} else {
			// Существующая карта
			serverID, err := uuid.Parse(clientCard.ServerID)
			if err != nil {
				s.log.Warn().Str("server_id", clientCard.ServerID).Msg("invalid server_id, skipping")
				continue
			}

			serverCard, found := serverCardsMap[serverID]
			if !found {
				// Карта есть у клиента, но нет на сервере (возможно, была удалена на другом устройстве)
				// Помечаем ее как удаленную для клиента
				clientCard.Deleted = true
				// cardsToClient = append(cardsToClient, clientCard.ToProto())
				continue
			}

			// Сравниваем checksum
			if clientCard.Checksum != serverCard.Checksum {
				// Checksum отличается - сравниваем время изменения
				clientChangeTime := clientCard.ChangeTime
				serverChangeTime := serverCard.UpdatedAt

				if clientChangeTime > serverChangeTime {
					// У клиента версия новее, обновляем на сервере
					serverCard.Number = clientCard.Number
					serverCard.Holder = clientCard.Holder
					serverCard.Expiry = clientCard.Expiry
					serverCard.CVV = clientCard.CVV
					serverCard.Metadata = clientCard.Metadata
					serverCard.Checksum = clientCard.Checksum
					serverCard.UpdatedAt = time.Now().Unix()
					if err := s.cardRepo.Update(ctx, serverCard); err != nil {
						s.log.Error().Err(err).Msgf("failed to update card with id %s", serverCard.ID)
						continue
					}
				} else {
					// У сервера версия новее, отправляем клиенту
					// responseCard := clientModel.Card{
					// 	LocalID:    localID,
					// 	ServerID:   serverCard.ID.String(),
					// 	Number:     serverCard.Number,
					// 	Holder:     serverCard.Holder,
					// 	Expiry:     serverCard.Expiry,
					// 	CVV:        serverCard.CVV,
					// 	Metadata:   serverCard.Metadata,
					// 	Checksum:   serverCard.Checksum,
					// 	ChangeTime: serverCard.UpdatedAt,
					// 	SyncTime:   time.Now().Unix(),
					// 	Deleted:    false,
					// }
					// cardsToClient = append(cardsToClient, responseCard.ToProto())
				}
			}
			// Удаляем из карты, чтобы потом найти те, что остались только на сервере
			delete(serverCardsMap, serverID)
		}
	}

	// Обрабатываем карты, которые остались только на сервере
	for _, serverCard := range serverCardsMap {
		// Ищем, есть ли у клиента карта с таким ServerID
		var clientHasIt bool
		for _, clientCard := range clientCards {
			if clientCard.ServerID == serverCard.ID.String() {
				clientHasIt = true
				// Если checksum отличается, отправляем полную карту
				if clientCard.Checksum != serverCard.Checksum {
					// responseCard := clientModel.Card{
					// 	LocalID:    clientCard.LocalID,
					// 	ServerID:   serverCard.ID.String(),
					// 	Number:     serverCard.Number,
					// 	Holder:     serverCard.Holder,
					// 	Expiry:     serverCard.Expiry,
					// 	CVV:        serverCard.CVV,
					// 	Metadata:   serverCard.Metadata,
					// 	Checksum:   serverCard.Checksum,
					// 	ChangeTime: serverCard.UpdatedAt,
					// 	SyncTime:   time.Now().Unix(),
					// 	Deleted:    false,
					// }
					// cardsToClient = append(cardsToClient, responseCard.ToProto())
				}
				break
			}
		}

		if !clientHasIt {
			// У клиента такой карты нет, отправляем ему
			// responseCard := clientModel.Card{
			// 	LocalID:    "", // Клиент должен будет присвоить свой local_id
			// 	ServerID:   serverCard.ID.String(),
			// 	Number:     serverCard.Number,
			// 	Holder:     serverCard.Holder,
			// 	Expiry:     serverCard.Expiry,
			// 	CVV:        serverCard.CVV,
			// 	Metadata:   serverCard.Metadata,
			// 	Checksum:   serverCard.Checksum,
			// 	ChangeTime: serverCard.UpdatedAt,
			// 	SyncTime:   time.Now().Unix(),
			// 	Deleted:    false,
			// }
			// cardsToClient = append(cardsToClient, responseCard.ToProto())
		}
	}

	s.log.Info().Msgf("Sending %d cards back to client", len(cardsToClient))
	return pb.GetCardsResponse_builder{Cards: cardsToClient}.Build(), nil
}
