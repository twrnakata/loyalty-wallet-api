package port

import "github.com/twrnakata/loyalty-wallet-api/internal/domain"

type UserRepository interface {
	Create(user *domain.User) error
	FindByEmail(email string) (*domain.User, error)
	FindByID(id uint) (*domain.User, error)
}

type WalletRepository interface {
	Create(wallet *domain.Wallet) error
	FindByUserID(userID uint) (*domain.Wallet, error)
	Update(wallet *domain.Wallet) error
}

type TransactionRepository interface {
	Create(tx *domain.Transaction) error
	FindByIdempotencyKey(walletID uint, key string) (*domain.Transaction, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenService interface {
	Issue(userID uint, email string) (string, error)
	Parse(token string) (userID uint, err error)
}
