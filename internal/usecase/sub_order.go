package usecase

import (
	"context"
	"fmt"

	"github.com/slipynil/avito-kitchen/internal/model"
	"github.com/slipynil/avito-kitchen/internal/repository"
)

type SubOrderUseCase struct {
	repo repository.SubOrderRepository
}

func NewSubOrderUseCase(repo repository.SubOrderRepository) *SubOrderUseCase {
	return &SubOrderUseCase{repo: repo}
}

// GetPendingOrders возвращает новые подзаказы ресторана со статусом pending.
func (uc *SubOrderUseCase) GetPendingOrders(ctx context.Context, restaurantID int) ([]model.SubOrder, error) {
	return uc.repo.GetByRestaurant(ctx, restaurantID, string(model.SubOrderStatusPending))
}

// UpdateStatus обновляет статус подзаказа с проверкой владения и валидацией статуса.
func (uc *SubOrderUseCase) UpdateStatus(
	ctx context.Context,
	subOrderID int,
	restaurantID int,
	status model.SubOrderStatus,
) (model.SubOrder, error) {
	switch status {
	case model.SubOrderStatusAccepted,
		model.SubOrderStatusPreparing,
		model.SubOrderStatusReady,
		model.SubOrderStatusCancelled:
		// допустимый статус
	default:
		return model.SubOrder{}, fmt.Errorf("invalid status: %s", status)
	}

	subOrder, err := uc.repo.GetByID(ctx, subOrderID)
	if err != nil {
		return model.SubOrder{}, err
	}

	if subOrder.RestaurantID != restaurantID {
		return model.SubOrder{}, fmt.Errorf("forbidden: sub order belongs to another restaurant")
	}

	return uc.repo.UpdateStatus(ctx, subOrderID, status)
}
