package services

import (
	"gophKeeper/internal/config"
	"net"

	pb "gophKeeper/internal/proto"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
)

// Server представляет собой обертку для gRPC-сервера.
type Server struct {
	server  *grpc.Server
	log     zerolog.Logger
	address string
	grpc_health_v1.UnimplementedHealthServer
}

// New создает новый gRPC-сервер.
// Он принимает логгер, реализацию KeeperService и адрес сервера.
func New(log zerolog.Logger, authService pb.AuthServiceServer, cfg config.Config) (*Server, error) {
	// Загружаем учетные данные TLS для сервера.
	// TODO: пути к файлам сертификатов лучше вынести в конфигурацию.
	creds, err := credentials.NewServerTLSFromFile("certs/server.crt", "certs/server.key")
	if err != nil {
		return nil, err
	}

	// Здесь можно добавить interceptors для логирования, аутентификации и т.д.
	s := grpc.NewServer(
		grpc.Creds(creds),
	)

	healthServer := health.NewServer()
    grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	
	// Регистрируем нашу реализацию сервиса на gRPC-сервере.
	pb.RegisterAuthServiceServer(s, authService)
	// TODO: Зарегистрировать здесь остальные сервисы (Device, Password и т.д.), когда они будут реализованы.

	// Условно регистрируем reflection service на gRPC-сервере.
	if cfg.EnableReflection {
		reflection.Register(s)
	}

	return &Server{
		server:  s,
		log:     log,
		address: cfg.ServerAddress,
	}, nil
}

// Start запускает gRPC-сервер.
func (s *Server) Start() error {
	listen, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	s.log.Info().Str("address", s.address).Msg("Starting gRPC server")
	return s.server.Serve(listen)
}

// Stop плавно останавливает gRPC-сервер.
func (s *Server) Stop() {
	s.log.Info().Msg("Stopping gRPC server")
	s.server.GracefulStop()
	s.log.Info().Msg("Server gracefully stopped")
}
