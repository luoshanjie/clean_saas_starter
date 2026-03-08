package main

import (
	"context"
	"log"

	"service/internal/bootstrap"
)

var (
	gitSHA    = "dev"
	buildTime = "dev"
)

// @title           Service API
// @version         0.1.0
// @description     Service API (development docs).
// @BasePath        /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
func main() {
	if err := bootstrap.Run(context.Background()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
