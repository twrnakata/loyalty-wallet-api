package main

import (
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/twrnakata/loyalty-wallet-api/internal/adapter/auth"
	httpadapter "github.com/twrnakata/loyalty-wallet-api/internal/adapter/http"
	"github.com/twrnakata/loyalty-wallet-api/internal/adapter/persistence"
	"github.com/twrnakata/loyalty-wallet-api/internal/service"
	"gorm.io/gorm"
)

func main() {
	dsn := env("DB_PATH", "loyalty.db")
	secret := env("JWT_SECRET", "dev-secret-change-me")
	addr := env("ADDR", ":3000")

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err := persistence.AutoMigrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	tokens := auth.NewJWTService(secret, 24*time.Hour)
	appSvc := service.New(
		persistence.NewUserRepo(db),
		persistence.NewWalletRepo(db),
		persistence.NewTxRepo(db),
		auth.BcryptHasher{},
		tokens,
	)

	fiberApp := fiber.New()
	httpadapter.NewServer(appSvc, tokens).Mount(fiberApp)

	log.Printf("loyalty-wallet-api listening on %s", addr)
	if err := fiberApp.Listen(addr); err != nil {
		log.Fatal(err)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
