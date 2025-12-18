package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims определяет структуру данных, которые будут храниться в JWT.
type Claims struct {
	jwt.RegisteredClaims
	UserID   uuid.UUID `json:"user_id"`
	DeviceID string    `json:"device_id"`
}

// JWTService предоставляет методы для работы с JWT токенами.
type JWTService struct {
	secret []byte
}

// NewJWTService создает новый экземпляр JWT сервиса.
func NewJWTService(secret []byte) *JWTService {
	return &JWTService{
		secret: secret,
	}
}

// GenerateToken генерирует JWT токен для указанного пользователя и устройства.
func (s *JWTService) GenerateToken(userID uuid.UUID, deviceID string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		UserID:   userID,
		DeviceID: deviceID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken валидирует JWT токен и возвращает claims.
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("failed to extract claims")
	}

	return claims, nil
}
