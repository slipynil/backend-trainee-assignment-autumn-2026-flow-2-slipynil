package usecase

import (
	"context"
	"fmt"

	"github.com/slipynil/avito-kitchen/internal/model"
	"github.com/slipynil/avito-kitchen/internal/repository"
)

type RestaurantUseCase struct {
	restaurantRepo repository.RestaurantRepository
	productRepo    repository.ProductRepository
}

func NewRestaurantUseCase(
	restaurantRepo repository.RestaurantRepository,
	productRepo repository.ProductRepository,
) *RestaurantUseCase {
	return &RestaurantUseCase{
		restaurantRepo: restaurantRepo,
		productRepo:    productRepo,
	}
}

// GetAll возвращает все активные рестораны.
func (uc *RestaurantUseCase) GetAll(ctx context.Context) ([]model.Restaurant, error) {
	return uc.restaurantRepo.GetAll(ctx)
}

// GetByID возвращает ресторан по его ID.
func (uc *RestaurantUseCase) GetByID(ctx context.Context, id int) (model.Restaurant, error) {
	return uc.restaurantRepo.GetByID(ctx, id)
}

// GetMenu возвращает меню ресторана (продукты ресторана).
func (uc *RestaurantUseCase) GetMenu(ctx context.Context, restaurantID int) ([]model.Product, error) {
	return uc.productRepo.GetByRestaurant(ctx, restaurantID)
}

// ValidateAPIKey проверяет API ключ ресторана и возвращает restaurantID.
func (uc *RestaurantUseCase) ValidateAPIKey(ctx context.Context, apiKey string) (int, error) {
	if apiKey == "" {
		return 0, fmt.Errorf("unauthorized: empty api key")
	}

	restaurant, err := uc.restaurantRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return 0, fmt.Errorf("unauthorized: invalid api key: %w", err)
	}

	return restaurant.ID, nil
}
