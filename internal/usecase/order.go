package usecase

import (
	"context"
	"fmt"
	"strconv"

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

// GetOrderStatus возвращает статус заказа по его id
func (orderUC *OrderUseCase) GetOrderStatus(ctx context.Context, orderIDstr, userIDstr string) (model.OrderStatus, error) {

	orderID, err := strconv.Atoi(orderIDstr)
	if err != nil {
		return model.OrderStatus(""), err
	}

	userID, err := strconv.Atoi(userIDstr)
	if err != nil {
		return model.OrderStatus(""), err
	}

	// 1. Получаем заказ по id
	order, err := orderUC.repo.GetByID(ctx, orderID)
	if err != nil {
		return model.OrderStatus(""), err
	}

	// 2. Проверяем, принадлежит ли заказ пользователю
	if order.ClientID != userID {
		return model.OrderStatus(""), fmt.Errorf("forbidden: you have no rights to update this order")
	}

	// 3. Возвращаем статус заказа
	return order.Status, nil
}

// UpdateOrderStatus обновляет статус заказа
func (orderUC *OrderUseCase) UpdateOrderStatus(
	ctx context.Context,
	orderIDstr string,
	userIDstr string,
	status model.OrderStatus,
) (model.Order, error) {

	orderID, err := strconv.Atoi(orderIDstr)
	if err != nil {
		return model.Order{}, err
	}

	userID, err := strconv.Atoi(userIDstr)
	if err != nil {
		return model.Order{}, err
	}

	// 1. Получаем заказ из базы
	order, err := orderUC.repo.GetByID(ctx, orderID)
	if err != nil {
		return model.Order{}, err // вернет ошибку, если заказ не найден
	}

	// 2. Проверяем, принадлежит ли заказ пользователю
	if order.ClientID != userID {
		return model.Order{}, fmt.Errorf("forbidden: you have no rights to update this order")
	}

	// 3. Обновляем статус
	return orderUC.repo.UpdateStatus(ctx, order.ID, status)
}

// GetOrderWithDetails возвращает заказ со всеми подзаказами через GetByIDWithSubOrders.
// Репозиторий уже группирует позиции по ресторанам при создании,
// здесь просто прокидываем вызов.
func (orderUC *OrderUseCase) GetOrderWithDetails(ctx context.Context, orderID int) (model.Order, error) {
	return orderUC.repo.GetByIDWithSubOrders(ctx, orderID)
}
