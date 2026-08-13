# loyalty-wallet-api

Personal demo API: register/login + points wallet (top-up / redeem) with idempotent redeem.

This is a **personal portfolio project**. It is not affiliated with any employer or client system.

## Stack

- Go + Fiber
- GORM + SQLite (pure Go driver via `glebarez/sqlite`)
- JWT + bcrypt
- Hexagonal-style layout (`domain` / `service` / `adapter`)
- Unit tests on the service layer (TDD)

## Run

```bash
go test ./...
go run ./cmd/api
```

Server listens on `:3000` by default (`ADDR`, `DB_PATH`, `JWT_SECRET` env vars supported).

## API

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| GET | `/health` | no | health check |
| POST | `/auth/register` | no | `{ "email", "password" }` |
| POST | `/auth/login` | no | returns `access_token` |
| POST | `/wallets` | Bearer | create wallet (balance 0) |
| GET | `/wallets/me` | Bearer | current balance |
| POST | `/wallets/me/topup` | Bearer | `{ "amount": 100 }` |
| POST | `/wallets/me/redeem` | Bearer | `{ "amount": 40 }` + header `Idempotency-Key` |

## Quick curl

```bash
# register + login
curl -s -X POST localhost:3000/auth/register -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"secret123"}'
TOKEN=$(curl -s -X POST localhost:3000/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"demo@example.com","password":"secret123"}' | jq -r .access_token)

# wallet flow
curl -s -X POST localhost:3000/wallets -H "Authorization: Bearer $TOKEN"
curl -s -X POST localhost:3000/wallets/me/topup -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"amount":100}'
curl -s -X POST localhost:3000/wallets/me/redeem -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-1' \
  -d '{"amount":40}'
# replay same key — balance must not decrease again
curl -s -X POST localhost:3000/wallets/me/redeem -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-1' \
  -d '{"amount":40}'
```

## Rules

- Amount must be `> 0`
- Balance never goes negative
- Redeem requires `Idempotency-Key`; same key returns the original transaction without double-charging
