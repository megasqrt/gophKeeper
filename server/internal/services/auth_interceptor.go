package services

import (
	"context"
	"strings"

	model "gophKeeper/pkg/grpchelper"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor проверяет JWT токен и добавляет userID в контекст.
func AuthInterceptor(log zerolog.Logger, jwtService *JWTService) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if info.FullMethod == "/gophkeeper.AuthService/Login" ||
			info.FullMethod == "/gophkeeper.AuthService/Register" {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Warn().Msg("No metadata in context")
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			log.Warn().Msg("No authorization header")
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Извлекаем токен из заголовка "Bearer <token>"
		authHeader := authHeaders[0]
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn().Str("header", authHeader).Msg("Invalid authorization header format")
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to validate JWT token")
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		userID := claims.UserID
		deviceID, parseDeviceIDErr := uuid.Parse(claims.DeviceID)
		if parseDeviceIDErr != nil {
			log.Warn().Err(parseDeviceIDErr).Str("device_id_str", claims.DeviceID).Msg("Failed to parse device_id from token")
			return nil, status.Error(codes.Unauthenticated, "invalid token: failed to parse device_id")
		}

		deviceIDHeaders := md.Get("x-device-id")
		if len(deviceIDHeaders) > 0 && deviceIDHeaders[0] != "" {
			headerDeviceID, err := uuid.Parse(deviceIDHeaders[0])
			if err == nil {
				if headerDeviceID != deviceID {
					log.Debug().Str("token_device_id", deviceID.String()).Str("header_device_id", headerDeviceID.String()).Msg("DeviceID from header differs from token, using header")
					deviceID = headerDeviceID
				}
			} else {
				log.Warn().Err(err).Str("header_device_id", deviceIDHeaders[0]).Msg("Failed to parse deviceID from header, using token deviceID")
			}
		}

		ctx = context.WithValue(ctx, model.UserKey, userID)
		ctx = context.WithValue(ctx, model.DeviceIDKey, deviceID)

		return handler(ctx, req)
	}
}
