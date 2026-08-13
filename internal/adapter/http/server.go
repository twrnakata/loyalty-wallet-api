package httpadapter

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/twrnakata/loyalty-wallet-api/internal/domain"
	"github.com/twrnakata/loyalty-wallet-api/internal/port"
	"github.com/twrnakata/loyalty-wallet-api/internal/service"
)

type Server struct {
	app    *service.App
	tokens port.TokenService
}

func NewServer(app *service.App, tokens port.TokenService) *Server {
	return &Server{app: app, tokens: tokens}
}

func (s *Server) Mount(r fiber.Router) {
	r.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	auth := r.Group("/auth")
	auth.Post("/register", s.register)
	auth.Post("/login", s.login)

	wallets := r.Group("/wallets", s.authRequired)
	wallets.Post("/", s.createWallet)
	wallets.Get("/me", s.getWallet)
	wallets.Post("/me/topup", s.topUp)
	wallets.Post("/me/redeem", s.redeem)
}

func (s *Server) authRequired(c *fiber.Ctx) error {
	h := c.Get("Authorization")
	if len(h) < 8 || h[:7] != "Bearer " {
		return writeErr(c, domain.ErrUnauthorized)
	}
	uid, err := s.tokens.Parse(h[7:])
	if err != nil {
		return writeErr(c, domain.ErrUnauthorized)
	}
	c.Locals("userID", uid)
	return c.Next()
}

func userID(c *fiber.Ctx) uint {
	v, _ := c.Locals("userID").(uint)
	return v
}

type credsReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) register(c *fiber.Ctx) error {
	var req credsReq
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, domain.ErrInvalidInput)
	}
	u, err := s.app.Register(req.Email, req.Password)
	if err != nil {
		return writeErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": u.ID, "email": u.Email,
	})
}

func (s *Server) login(c *fiber.Ctx) error {
	var req credsReq
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, domain.ErrInvalidInput)
	}
	token, u, err := s.app.Login(req.Email, req.Password)
	if err != nil {
		return writeErr(c, err)
	}
	return c.JSON(fiber.Map{
		"access_token": token,
		"token_type":   "Bearer",
		"user":         fiber.Map{"id": u.ID, "email": u.Email},
	})
}

func (s *Server) createWallet(c *fiber.Ctx) error {
	w, err := s.app.CreateWallet(userID(c))
	if err != nil {
		return writeErr(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(walletJSON(w))
}

func (s *Server) getWallet(c *fiber.Ctx) error {
	w, err := s.app.GetWallet(userID(c))
	if err != nil {
		return writeErr(c, err)
	}
	return c.JSON(walletJSON(w))
}

type amountReq struct {
	Amount int64 `json:"amount"`
}

func (s *Server) topUp(c *fiber.Ctx) error {
	var req amountReq
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, domain.ErrInvalidInput)
	}
	w, err := s.app.TopUp(userID(c), req.Amount)
	if err != nil {
		return writeErr(c, err)
	}
	return c.JSON(walletJSON(w))
}

func (s *Server) redeem(c *fiber.Ctx) error {
	var req amountReq
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, domain.ErrInvalidInput)
	}
	key := c.Get("Idempotency-Key")
	w, tx, err := s.app.Redeem(userID(c), req.Amount, key)
	if err != nil {
		return writeErr(c, err)
	}
	return c.JSON(fiber.Map{
		"wallet": walletJSON(w),
		"transaction": fiber.Map{
			"id":              tx.ID,
			"type":            tx.Type,
			"amount":          tx.Amount,
			"balance_after":   tx.BalanceAfter,
			"idempotency_key": tx.IdempotencyKey,
		},
	})
}

func walletJSON(w *domain.Wallet) fiber.Map {
	return fiber.Map{
		"id": w.ID, "user_id": w.UserID, "balance": w.Balance,
	}
}

func writeErr(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	msg := err.Error()
	switch {
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrMissingIdempotencyKey):
		status = fiber.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized),
		errors.Is(err, domain.ErrInvalidCredentials):
		status = fiber.StatusUnauthorized
	case errors.Is(err, domain.ErrNotFound):
		status = fiber.StatusNotFound
	case errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrWalletExists),
		errors.Is(err, domain.ErrConflict):
		status = fiber.StatusConflict
	case errors.Is(err, domain.ErrInsufficientBalance):
		status = fiber.StatusUnprocessableEntity
	}
	return c.Status(status).JSON(fiber.Map{"error": msg})
}
