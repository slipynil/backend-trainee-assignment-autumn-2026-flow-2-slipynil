package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/slipynil/avito-kitchen/internal/api"
	"github.com/slipynil/avito-kitchen/internal/database"
	"github.com/slipynil/avito-kitchen/internal/delivery"
	"github.com/slipynil/avito-kitchen/internal/repository"
	"github.com/slipynil/avito-kitchen/internal/usecase"
)

func main() {
	dbConnStr := os.Getenv("DB_CONN")
	if len(dbConnStr) == 0 {
		log.Fatal("DB_CONN is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	db, err := database.Connect(ctx, dbConnStr)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close(context.Background())

	// Репозитории
	restaurantRepo := repository.NewRestaurantRepository(db)
	productRepo := repository.NewProductRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	subOrderRepo := repository.NewSubOrderRepository(db)

	// Use cases
	restaurantUC := usecase.NewRestaurantUseCase(*restaurantRepo, *productRepo)
	productUC := usecase.NewProductUseCase(*productRepo)
	orderUC := usecase.NewOrderUseCase(*orderRepo)
	subOrderUC := usecase.NewSubOrderUseCase(*subOrderRepo)

	// Handler
	h := delivery.NewHandler(productUC, orderUC, restaurantUC, subOrderUC)

	// Echo, регистрация маршрутов.
	// Партнёрские операции защищены RestaurantAuthMiddleware.
	e := echo.New()
	e.HideBanner = true

	restaurantAuth := delivery.RestaurantAuthMiddleware(restaurantUC)
	api.RegisterHandlersWithOptions(e, h, api.RegisterHandlersOptions{
		OperationMiddlewares: map[string][]echo.MiddlewareFunc{
			"GetRestaurantOrders":  {restaurantAuth},
			"UpdateSubOrderStatus": {restaurantAuth},
			"GetMyProducts":        {restaurantAuth},
			"CreateProduct":        {restaurantAuth},
			"UpdateProduct":        {restaurantAuth},
			"DeleteProduct":        {restaurantAuth},
		},
	})

	// Запуск сервера + graceful shutdown.
	serverErr := make(chan error, 1)
	go func() {
		if err := e.Start(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()
	log.Printf("server listening on :%s", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("received signal %s, shutting down", sig)
	case err := <-serverErr:
		log.Fatalf("server failed: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}
