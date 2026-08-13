package domain

import "time"

type TxType string

const (
	TxTopUp  TxType = "topup"
	TxRedeem TxType = "redeem"
)

type Wallet struct {
	ID        uint
	UserID    uint
	Balance   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Transaction struct {
	ID             uint
	WalletID       uint
	Type           TxType
	Amount         int64
	BalanceAfter   int64
	IdempotencyKey string
	CreatedAt      time.Time
}
