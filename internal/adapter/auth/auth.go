package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/twrnakata/loyalty-wallet-api/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct{}

func (BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func (BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

type JWTService struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTService(secret string, ttl time.Duration) *JWTService {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &JWTService{secret: []byte(secret), ttl: ttl}
}

type claims struct {
	UserID uint   `json:"uid"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (j *JWTService) Issue(userID uint, email string) (string, error) {
	c := claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(j.secret)
}

func (j *JWTService) Parse(token string) (uint, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		return j.secret, nil
	})
	if err != nil || !parsed.Valid {
		return 0, domain.ErrUnauthorized
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || c.UserID == 0 {
		return 0, domain.ErrUnauthorized
	}
	return c.UserID, nil
}
