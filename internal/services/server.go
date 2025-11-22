package services

import (
	"net"
	"gophKeeper/internal/config"

	pb "gophKeeper/internal/proto"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server представляет собой обертку для gRPC-сервера.
type Server struct {
	server *grpc.Server
	log        zerolog.Logger
	address    string
}

// New создает новый gRPC-сервер.
// Он принимает логгер, реализацию KeeperService и адрес сервера.
func New(log zerolog.Logger, keeperService pb.KeeperServiceServer, cfg config.Config) (*Server, error) {
	// Здесь можно добавить interceptors для логирования, аутентификации и т.д.
	s := grpc.NewServer()

	// Регистрируем нашу реализацию сервиса на gRPC-сервере.
	pb.RegisterKeeperServiceServer(s, keeperService)

	// Условно регистрируем reflection service на gRPC-сервере.
	if cfg.EnableReflection {
		reflection.Register(s)
	}

	return &Server{
		server: s,
		log:        log,
		address:    cfg.ServerAddress,
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

		// listen, err := net.Listen("tcp", cfg.GRPCServerAddress)
		// if err != nil {
		// 	log.Fatal().Err(err).Msg("Failed to listen for gRPC")
		// }
		// s := grpc.NewServer(grpc.StreamInterceptor(middlewares.GrpcCheckMiddleware(cfg.TrustedSubnet)))
		// pb.RegisterMetricsServer(s, services.NewMetricGRPCServer(storage, log))
		// log.Info().Str("address", cfg.GRPCServerAddress).Msg("Starting gRPC server")
		// if err := s.Serve(listen); err != nil {
		// 	log.Fatal().Err(err).Msg("gRPC server error")
		// }
		//	if err := gRPCServer.Start(); err != nil {

// Stop плавно останавливает gRPC-сервер.
func (s *Server) Stop() {
	s.log.Info().Msg("Stopping gRPC server")
	s.server.GracefulStop()
	s.log.Info().Msg("Server gracefully stopped")
}
