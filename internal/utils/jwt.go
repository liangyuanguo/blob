package utils

import (
	"errors"
	"liangyuanguo/aw/blob/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type JWTUtil struct {
	secretKey string
}

func NewJWTUtil() *JWTUtil {
	return &JWTUtil{secretKey: config.Config.Jwt.Secret}
}

// CustomClaims 自定义Claims结构
type CustomClaims struct {
	Meta any `json:"meta"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT token
func (j *JWTUtil) GenerateToken(meta any, expiresIn time.Duration) (string, error) {
	claims := CustomClaims{
		Meta: meta,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "any",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ParseToken 解析并验证JWT token
func (j *JWTUtil) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
