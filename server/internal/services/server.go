package services

import (
	"gophKeeper/server/internal/config"
	"net"

	pb "gophKeeper/pkg/proto"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
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
func New(
	log zerolog.Logger,
	authService pb.AuthServiceServer,
	noteService pb.NoteServiceServer,
	cardService pb.CardServiceServer,
	passService pb.PasswordServiceServer,
	fileService pb.FileServiceServer,
	cfg config.Config) (*Server, error) {
	creds, err := credentials.NewServerTLSFromFile("certs/server.crt", "certs/server.key")
	if err != nil {
		return nil, err
	}

	// Создаем interceptor для аутентификации
	authInterceptor := AuthInterceptor(log, []byte(cfg.HashKey))

	s := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(authInterceptor),
	)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Регистрируем нашу реализацию сервиса на gRPC-сервере.
	pb.RegisterAuthServiceServer(s, authService)
	pb.RegisterNoteServiceServer(s, noteService)
	pb.RegisterCardServiceServer(s, cardService)
	pb.RegisterPasswordServiceServer(s, passService)
	pb.RegisterFileServiceServer(s, fileService)

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
