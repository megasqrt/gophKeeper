package services

import (
	"context"
	"fmt"
	"strings"

	model "gophKeeper/pkg/grpchelper"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor проверяет JWT токен и добавляет userID в контекст.
func AuthInterceptor(log zerolog.Logger, jwtSecret []byte) grpc.UnaryServerInterceptor {
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

		token, parseErr := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, status.Errorf(codes.Unauthenticated, "unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if parseErr != nil {
			log.Warn().Err(parseErr).Msg("Failed to parse JWT token")
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		if !token.Valid {
			log.Warn().Msg("Invalid token")
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Извлекаем claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Warn().Msg("Failed to extract claims")
			return nil, status.Error(codes.Unauthenticated, "invalid token claims")
		}

		userIDValue, exists := claims["user_id"]
		if !exists {
			log.Warn().Interface("claims", claims).Msg("No user_id in token claims")
			return nil, status.Error(codes.Unauthenticated, "invalid token: missing user_id")
		}

		userIDStr, ok := userIDValue.(string)
		if !ok {
			// Если не строка, пробуем преобразовать через fmt
			userIDStr = fmt.Sprintf("%v", userIDValue)
			log.Debug().Str("user_id_raw", userIDStr).Msg("Converted user_id to string")
		}

		userID, parseUserIDErr := uuid.Parse(userIDStr)
		if parseUserIDErr != nil {
			log.Warn().Err(parseUserIDErr).Str("user_id_str", userIDStr).Interface("claims", claims).Msg("Failed to parse user_id from token")
			return nil, status.Error(codes.Unauthenticated, "invalid token: failed to parse user_id")
		}

		deviceIDValue, exists := claims["device_id"]
		if !exists {
			log.Warn().Interface("claims", claims).Msg("No device_id in token claims")
			return nil, status.Error(codes.Unauthenticated, "invalid token: missing device_id")
		}
		deviceIDStr, ok := deviceIDValue.(string)
		if !ok {
			deviceIDStr = fmt.Sprintf("%v", deviceIDValue)
			log.Debug().Str("device_id_raw", deviceIDStr).Msg("Converted device_id to string")
		}

		// Парсим deviceID в UUID
		deviceID, parseDeviceIDErr := uuid.Parse(deviceIDStr)
		if parseDeviceIDErr != nil {
			log.Warn().Err(parseDeviceIDErr).Str("device_id_str", deviceIDStr).Interface("claims", claims).Msg("Failed to parse device_id from token")
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
