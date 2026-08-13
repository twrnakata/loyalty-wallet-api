package service

import (
	"strings"

	"github.com/twrnakata/loyalty-wallet-api/internal/domain"
	"github.com/twrnakata/loyalty-wallet-api/internal/port"
)

type App struct {
	users    port.UserRepository
	wallets  port.WalletRepository
	txs      port.TransactionRepository
	hasher   port.PasswordHasher
	tokens   port.TokenService
}

func New(
	users port.UserRepository,
	wallets port.WalletRepository,
	txs port.TransactionRepository,
	hasher port.PasswordHasher,
	tokens port.TokenService,
) *App {
	return &App{users: users, wallets: wallets, txs: txs, hasher: hasher, tokens: tokens}
}

func (a *App) Register(email, password string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(password) < 6 {
		return nil, domain.ErrInvalidInput
	}
	if _, err := a.users.FindByEmail(email); err == nil {
		return nil, domain.ErrEmailTaken
	} else if err != domain.ErrNotFound {
		return nil, err
	}
	hash, err := a.hasher.Hash(password)
	if err != nil {
		return nil, err
	}
	u := &domain.User{Email: email, PasswordHash: hash}
	if err := a.users.Create(u); err != nil {
		return nil, err
	}
	return u, nil
}

func (a *App) Login(email, password string) (token string, user *domain.User, err error) {
	email = strings.TrimSpace(strings.ToLower(email))
	u, err := a.users.FindByEmail(email)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", nil, domain.ErrInvalidCredentials
		}
		return "", nil, err
	}
	if err := a.hasher.Compare(u.PasswordHash, password); err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}
	token, err = a.tokens.Issue(u.ID, u.Email)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

func (a *App) CreateWallet(userID uint) (*domain.Wallet, error) {
	if _, err := a.wallets.FindByUserID(userID); err == nil {
		return nil, domain.ErrWalletExists
	} else if err != domain.ErrNotFound {
		return nil, err
	}
	w := &domain.Wallet{UserID: userID, Balance: 0}
	if err := a.wallets.Create(w); err != nil {
		return nil, err
	}
	return w, nil
}

func (a *App) GetWallet(userID uint) (*domain.Wallet, error) {
	return a.wallets.FindByUserID(userID)
}

func (a *App) TopUp(userID uint, amount int64) (*domain.Wallet, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	w, err := a.wallets.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	w.Balance += amount
	if err := a.wallets.Update(w); err != nil {
		return nil, err
	}
	tx := &domain.Transaction{
		WalletID:     w.ID,
		Type:         domain.TxTopUp,
		Amount:       amount,
		BalanceAfter: w.Balance,
	}
	if err := a.txs.Create(tx); err != nil {
		return nil, err
	}
	return w, nil
}

func (a *App) Redeem(userID uint, amount int64, idempotencyKey string) (*domain.Wallet, *domain.Transaction, error) {
	if amount <= 0 {
		return nil, nil, domain.ErrInvalidAmount
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, nil, domain.ErrMissingIdempotencyKey
	}
	w, err := a.wallets.FindByUserID(userID)
	if err != nil {
		return nil, nil, err
	}
	if existing, err := a.txs.FindByIdempotencyKey(w.ID, idempotencyKey); err == nil {
		return w, existing, nil
	} else if err != domain.ErrNotFound {
		return nil, nil, err
	}
	if w.Balance < amount {
		return nil, nil, domain.ErrInsufficientBalance
	}
	w.Balance -= amount
	if err := a.wallets.Update(w); err != nil {
		return nil, nil, err
	}
	tx := &domain.Transaction{
		WalletID:       w.ID,
		Type:           domain.TxRedeem,
		Amount:         amount,
		BalanceAfter:   w.Balance,
		IdempotencyKey: idempotencyKey,
	}
	if err := a.txs.Create(tx); err != nil {
		return nil, nil, err
	}
	return w, tx, nil
}
