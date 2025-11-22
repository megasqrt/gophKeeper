package grpc

import (
	pb "gophKeeper/internal/proto"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClient создает и возвращает нового gRPC клиента для KeeperService.
func NewClient(serverAddr string) (pb.KeeperServiceClient, *grpc.ClientConn) {
	// Устанавливаем соединение с сервером.
	// Используем insecure-соединение для простоты, в продакшене нужно использовать TLS.
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	// Создаем нового клиента.
	client := pb.NewKeeperServiceClient(conn)
	return client, conn
}
