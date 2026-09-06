package usecase

import (
	"context"
	"fmt"

	"github.com/slipynil/avito-kitchen/internal/model"
	"github.com/slipynil/avito-kitchen/internal/repository"
)

type ProductUseCase struct {
	repo repository.ProductRepository
}

func NewProductUseCase(repo repository.ProductRepository) *ProductUseCase {
	return &ProductUseCase{repo: repo}
}

// GetProducts возвращает все продукты
func (uc *ProductUseCase) GetProducts(ctx context.Context) ([]model.Product, error) {
	return uc.repo.GetAll(ctx)
}

// GetByRestaurant возвращает продукты ресторана.
func (uc *ProductUseCase) GetByRestaurant(ctx context.Context, restaurantID int) ([]model.Product, error) {
	return uc.repo.GetByRestaurant(ctx, restaurantID)
}

// CreateProduct создаёт продукт с принудительной привязкой к ресторану.
func (uc *ProductUseCase) CreateProduct(ctx context.Context, product model.Product, restaurantID int) (model.Product, error) {
	product.RestaurantID = restaurantID
	return uc.repo.Create(ctx, product)
}

// UpdateProduct обновляет продукт с проверкой владения.
func (uc *ProductUseCase) UpdateProduct(ctx context.Context, product model.Product, restaurantID int) (model.Product, error) {
	existing, err := uc.repo.GetByID(ctx, product.ID)
	if err != nil {
		return model.Product{}, err
	}

	if existing.RestaurantID != restaurantID {
		return model.Product{}, fmt.Errorf("forbidden: product belongs to another restaurant")
	}

	product.RestaurantID = restaurantID
	return uc.repo.Update(ctx, product)
}

// DeleteProduct удаляет продукт с проверкой владения.
func (uc *ProductUseCase) DeleteProduct(ctx context.Context, productID int, restaurantID int) error {
	existing, err := uc.repo.GetByID(ctx, productID)
	if err != nil {
		return err
	}

	if existing.RestaurantID != restaurantID {
		return fmt.Errorf("forbidden: product belongs to another restaurant")
	}

	return uc.repo.Delete(ctx, productID)
}
