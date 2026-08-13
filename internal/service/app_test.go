package service_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/twrnakata/loyalty-wallet-api/internal/domain"
	"github.com/twrnakata/loyalty-wallet-api/internal/service"
)

type memUsers struct {
	mu     sync.Mutex
	byID   map[uint]*domain.User
	byMail map[string]*domain.User
	seq    uint
}

func newMemUsers() *memUsers {
	return &memUsers{byID: map[uint]*domain.User{}, byMail: map[string]*domain.User{}}
}

func (m *memUsers) Create(u *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byMail[u.Email]; ok {
		return domain.ErrEmailTaken
	}
	m.seq++
	cp := *u
	cp.ID = m.seq
	m.byID[cp.ID] = &cp
	m.byMail[cp.Email] = &cp
	u.ID = cp.ID
	return nil
}

func (m *memUsers) FindByEmail(email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byMail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memUsers) FindByID(id uint) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

type memWallets struct {
	mu    sync.Mutex
	byUID map[uint]*domain.Wallet
	seq   uint
}

func newMemWallets() *memWallets {
	return &memWallets{byUID: map[uint]*domain.Wallet{}}
}

func (m *memWallets) Create(w *domain.Wallet) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byUID[w.UserID]; ok {
		return domain.ErrWalletExists
	}
	m.seq++
	cp := *w
	cp.ID = m.seq
	m.byUID[cp.UserID] = &cp
	w.ID = cp.ID
	return nil
}

func (m *memWallets) FindByUserID(userID uint) (*domain.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.byUID[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *w
	return &cp, nil
}

func (m *memWallets) Update(w *domain.Wallet) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byUID[w.UserID]; !ok {
		return domain.ErrNotFound
	}
	cp := *w
	m.byUID[w.UserID] = &cp
	return nil
}

type memTx struct {
	mu   sync.Mutex
	byKey map[string]*domain.Transaction
	all  []*domain.Transaction
	seq  uint
}

func newMemTx() *memTx {
	return &memTx{byKey: map[string]*domain.Transaction{}}
}

func (m *memTx) Create(tx *domain.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tx.IdempotencyKey != "" {
		k := key(tx.WalletID, tx.IdempotencyKey)
		if _, ok := m.byKey[k]; ok {
			return domain.ErrConflict
		}
		m.seq++
		cp := *tx
		cp.ID = m.seq
		m.byKey[k] = &cp
		m.all = append(m.all, &cp)
		tx.ID = cp.ID
		return nil
	}
	m.seq++
	cp := *tx
	cp.ID = m.seq
	m.all = append(m.all, &cp)
	tx.ID = cp.ID
	return nil
}

func (m *memTx) FindByIdempotencyKey(walletID uint, idemKey string) (*domain.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, ok := m.byKey[key(walletID, idemKey)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *tx
	return &cp, nil
}

func key(walletID uint, idemKey string) string {
	return fmt.Sprintf("%d:%s", walletID, idemKey)
}

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) { return "hash:" + password, nil }
func (fakeHasher) Compare(hash, password string) error {
	if hash != "hash:"+password {
		return errors.New("mismatch")
	}
	return nil
}

type fakeToken struct{}

func (fakeToken) Issue(userID uint, email string) (string, error) {
	return "token-for-" + email, nil
}
func (fakeToken) Parse(token string) (uint, error) {
	return 0, domain.ErrUnauthorized
}

func newSvc() *service.App {
	return service.New(newMemUsers(), newMemWallets(), newMemTx(), fakeHasher{}, fakeToken{})
}

func TestRegister_Success(t *testing.T) {
	svc := newSvc()
	u, err := svc.Register("a@example.com", "secret123")
	require.NoError(t, err)
	require.Equal(t, "a@example.com", u.Email)
	require.NotZero(t, u.ID)
	require.Equal(t, "hash:secret123", u.PasswordHash)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newSvc()
	_, err := svc.Register("a@example.com", "secret123")
	require.NoError(t, err)
	_, err = svc.Register("a@example.com", "otherpassword")
	require.ErrorIs(t, err, domain.ErrEmailTaken)
}

func TestRegister_InvalidInput(t *testing.T) {
	svc := newSvc()
	_, err := svc.Register("", "secret123")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
	_, err = svc.Register("a@example.com", "123")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestLogin_Success(t *testing.T) {
	svc := newSvc()
	_, err := svc.Register("a@example.com", "secret123")
	require.NoError(t, err)
	token, u, err := svc.Login("a@example.com", "secret123")
	require.NoError(t, err)
	require.Equal(t, "token-for-a@example.com", token)
	require.Equal(t, "a@example.com", u.Email)
}

func TestLogin_BadPassword(t *testing.T) {
	svc := newSvc()
	_, err := svc.Register("a@example.com", "secret123")
	require.NoError(t, err)
	_, _, err = svc.Login("a@example.com", "wrong")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestCreateWallet_AndGet(t *testing.T) {
	svc := newSvc()
	u, err := svc.Register("a@example.com", "secret123")
	require.NoError(t, err)
	w, err := svc.CreateWallet(u.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), w.Balance)
	got, err := svc.GetWallet(u.ID)
	require.NoError(t, err)
	require.Equal(t, w.ID, got.ID)
}

func TestCreateWallet_Duplicate(t *testing.T) {
	svc := newSvc()
	u, err := svc.Register("a@example.com", "secret123")
	require.NoError(t, err)
	_, err = svc.CreateWallet(u.ID)
	require.NoError(t, err)
	_, err = svc.CreateWallet(u.ID)
	require.ErrorIs(t, err, domain.ErrWalletExists)
}

func TestTopUp_Success(t *testing.T) {
	svc := newSvc()
	u, _ := svc.Register("a@example.com", "secret123")
	_, _ = svc.CreateWallet(u.ID)
	w, err := svc.TopUp(u.ID, 100)
	require.NoError(t, err)
	require.Equal(t, int64(100), w.Balance)
}

func TestTopUp_InvalidAmount(t *testing.T) {
	svc := newSvc()
	u, _ := svc.Register("a@example.com", "secret123")
	_, _ = svc.CreateWallet(u.ID)
	_, err := svc.TopUp(u.ID, 0)
	require.ErrorIs(t, err, domain.ErrInvalidAmount)
}

func TestRedeem_Success(t *testing.T) {
	svc := newSvc()
	u, _ := svc.Register("a@example.com", "secret123")
	_, _ = svc.CreateWallet(u.ID)
	_, _ = svc.TopUp(u.ID, 100)
	w, tx, err := svc.Redeem(u.ID, 40, "key-1")
	require.NoError(t, err)
	require.Equal(t, int64(60), w.Balance)
	require.Equal(t, domain.TxRedeem, tx.Type)
	require.Equal(t, int64(40), tx.Amount)
	require.Equal(t, "key-1", tx.IdempotencyKey)
}

func TestRedeem_InsufficientBalance(t *testing.T) {
	svc := newSvc()
	u, _ := svc.Register("a@example.com", "secret123")
	_, _ = svc.CreateWallet(u.ID)
	_, _ = svc.TopUp(u.ID, 10)
	_, _, err := svc.Redeem(u.ID, 40, "key-1")
	require.ErrorIs(t, err, domain.ErrInsufficientBalance)
}

func TestRedeem_RequiresIdempotencyKey(t *testing.T) {
	svc := newSvc()
	u, _ := svc.Register("a@example.com", "secret123")
	_, _ = svc.CreateWallet(u.ID)
	_, _ = svc.TopUp(u.ID, 100)
	_, _, err := svc.Redeem(u.ID, 10, "")
	require.ErrorIs(t, err, domain.ErrMissingIdempotencyKey)
}

func TestRedeem_IdempotentReplay(t *testing.T) {
	svc := newSvc()
	u, _ := svc.Register("a@example.com", "secret123")
	_, _ = svc.CreateWallet(u.ID)
	_, _ = svc.TopUp(u.ID, 100)

	w1, tx1, err := svc.Redeem(u.ID, 30, "same-key")
	require.NoError(t, err)
	w2, tx2, err := svc.Redeem(u.ID, 30, "same-key")
	require.NoError(t, err)

	require.Equal(t, w1.Balance, w2.Balance)
	require.Equal(t, int64(70), w2.Balance)
	require.Equal(t, tx1.ID, tx2.ID)
}
