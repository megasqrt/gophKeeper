package services

import (
	"context"
	"fmt"
	"strings"

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
		// Пропускаем аутентификацию для методов аутентификации
		if info.FullMethod == "/gophkeeper.AuthService/Login" ||
			info.FullMethod == "/gophkeeper.AuthService/Register" {
			return handler(ctx, req)
		}

		// Извлекаем токен из метаданных
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

		// Парсим и проверяем токен
		token, parseErr := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Проверяем метод подписи
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

		// Извлекаем userID из claims
		// UUID в JWT сериализуется как строка
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

		// Добавляем userID в контекст
		ctx = context.WithValue(ctx, "userID", userID)

		// Вызываем следующий обработчик
		return handler(ctx, req)
	}
}

