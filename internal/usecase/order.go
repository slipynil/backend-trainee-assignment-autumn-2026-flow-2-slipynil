package usecase

import (
	"context"

	"github.com/slipynil/avito-kitchen/internal/dto"
	"github.com/slipynil/avito-kitchen/internal/model"
	"github.com/slipynil/avito-kitchen/internal/repository"
)

type OrderUseCase struct {
	repo repository.OrderRepository
}

func NewOrderUseCase(repo repository.OrderRepository) *OrderUseCase {
	return &OrderUseCase{repo: repo}
}

// CreateOrder создает новый заказ
func (orderUC *OrderUseCase) CreateOrder(ctx context.Context, clientID int, deliveryAddress string, items []dto.OrderItem) (model.Order, error) {
	convertedItems := make([]model.OrderItem, len(items))

	for i, item := range items {
		convertedItems[i] = model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	return orderUC.repo.Create(ctx, clientID, deliveryAddress, convertedItems)
}

// GetOrderWithDetails возвращает заказ со всеми подзаказами через GetByIDWithSubOrders.
// Репозиторий уже группирует позиции по ресторанам при создании,
// здесь просто прокидываем вызов.
func (orderUC *OrderUseCase) GetOrderWithDetails(ctx context.Context, orderID int) (model.Order, error) {
	return orderUC.repo.GetByIDWithSubOrders(ctx, orderID)
}
