package persistence

import (
	"errors"
	"strings"
	"time"

	"github.com/twrnakata/loyalty-wallet-api/internal/domain"
	"gorm.io/gorm"
)

type UserModel struct {
	ID           uint   `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex;size:255;not null"`
	PasswordHash string `gorm:"not null"`
	CreatedAt    time.Time
}

func (UserModel) TableName() string { return "users" }

type WalletModel struct {
	ID        uint  `gorm:"primaryKey"`
	UserID    uint  `gorm:"uniqueIndex;not null"`
	Balance   int64 `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (WalletModel) TableName() string { return "wallets" }

type TransactionModel struct {
	ID             uint   `gorm:"primaryKey"`
	WalletID       uint   `gorm:"index;not null"`
	Type           string `gorm:"size:16;not null"`
	Amount         int64  `gorm:"not null"`
	BalanceAfter   int64  `gorm:"not null"`
	IdempotencyKey string `gorm:"size:128"`
	CreatedAt      time.Time
}

func (TransactionModel) TableName() string { return "transactions" }

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&UserModel{}, &WalletModel{}, &TransactionModel{}); err != nil {
		return err
	}
	// Unique (wallet_id, idempotency_key) when key is present — enforced in app + partial uniqueness via composite index.
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_tx_wallet_idem ON transactions(wallet_id, idempotency_key) WHERE idempotency_key != ''`).Error
}

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) Create(user *domain.User) error {
	m := UserModel{Email: user.Email, PasswordHash: user.PasswordHash}
	if err := r.db.Create(&m).Error; err != nil {
		if isUnique(err) {
			return domain.ErrEmailTaken
		}
		return err
	}
	user.ID = m.ID
	user.CreatedAt = m.CreatedAt
	return nil
}

func (r *UserRepo) FindByEmail(email string) (*domain.User, error) {
	var m UserModel
	if err := r.db.Where("email = ?", email).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.User{ID: m.ID, Email: m.Email, PasswordHash: m.PasswordHash, CreatedAt: m.CreatedAt}, nil
}

func (r *UserRepo) FindByID(id uint) (*domain.User, error) {
	var m UserModel
	if err := r.db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.User{ID: m.ID, Email: m.Email, PasswordHash: m.PasswordHash, CreatedAt: m.CreatedAt}, nil
}

type WalletRepo struct{ db *gorm.DB }

func NewWalletRepo(db *gorm.DB) *WalletRepo { return &WalletRepo{db: db} }

func (r *WalletRepo) Create(wallet *domain.Wallet) error {
	m := WalletModel{UserID: wallet.UserID, Balance: wallet.Balance}
	if err := r.db.Create(&m).Error; err != nil {
		if isUnique(err) {
			return domain.ErrWalletExists
		}
		return err
	}
	wallet.ID = m.ID
	wallet.CreatedAt = m.CreatedAt
	wallet.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *WalletRepo) FindByUserID(userID uint) (*domain.Wallet, error) {
	var m WalletModel
	if err := r.db.Where("user_id = ?", userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.Wallet{
		ID: m.ID, UserID: m.UserID, Balance: m.Balance,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *WalletRepo) Update(wallet *domain.Wallet) error {
	res := r.db.Model(&WalletModel{}).Where("id = ?", wallet.ID).Updates(map[string]any{
		"balance": wallet.Balance,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type TxRepo struct{ db *gorm.DB }

func NewTxRepo(db *gorm.DB) *TxRepo { return &TxRepo{db: db} }

func (r *TxRepo) Create(tx *domain.Transaction) error {
	m := TransactionModel{
		WalletID:       tx.WalletID,
		Type:           string(tx.Type),
		Amount:         tx.Amount,
		BalanceAfter:   tx.BalanceAfter,
		IdempotencyKey: tx.IdempotencyKey,
	}
	if err := r.db.Create(&m).Error; err != nil {
		if isUnique(err) {
			return domain.ErrConflict
		}
		return err
	}
	tx.ID = m.ID
	tx.CreatedAt = m.CreatedAt
	return nil
}

func (r *TxRepo) FindByIdempotencyKey(walletID uint, key string) (*domain.Transaction, error) {
	var m TransactionModel
	if err := r.db.Where("wallet_id = ? AND idempotency_key = ?", walletID, key).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.Transaction{
		ID: m.ID, WalletID: m.WalletID, Type: domain.TxType(m.Type),
		Amount: m.Amount, BalanceAfter: m.BalanceAfter,
		IdempotencyKey: m.IdempotencyKey, CreatedAt: m.CreatedAt,
	}, nil
}

func isUnique(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE") || strings.Contains(msg, "unique")
}
