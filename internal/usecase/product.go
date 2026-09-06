package usecase

import (
	"context"

	"github.com/slipynil/avito-kitchen/internal/model"
	"github.com/slipynil/avito-kitchen/internal/repository"
)

type ProductUseCase struct {
	repo repository.ProductRepository
}

// GetProducts возвращает все продукты
func (uc *ProductUseCase) GetProducts(ctx context.Context) ([]model.Product, error) {
	return uc.repo.GetAll(ctx)
}
