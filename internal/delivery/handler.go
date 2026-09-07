package delivery

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/slipynil/avito-kitchen/internal/api"
	"github.com/slipynil/avito-kitchen/internal/dto"
	"github.com/slipynil/avito-kitchen/internal/model"
	"github.com/slipynil/avito-kitchen/internal/usecase"
)

// Handler реализует api.ServerInterface (сгенерирован oapi-codegen, echo v4).
type Handler struct {
	productUC    *usecase.ProductUseCase
	orderUC      *usecase.OrderUseCase
	restaurantUC *usecase.RestaurantUseCase
	subOrderUC   *usecase.SubOrderUseCase
}

func NewHandler(
	productUC *usecase.ProductUseCase,
	orderUC *usecase.OrderUseCase,
	restaurantUC *usecase.RestaurantUseCase,
	subOrderUC *usecase.SubOrderUseCase,
) *Handler {
	return &Handler{
		productUC:    productUC,
		orderUC:      orderUC,
		restaurantUC: restaurantUC,
		subOrderUC:   subOrderUC,
	}
}

// Гарантируем соответствие интерфейсу на этапе компиляции.
var _ api.ServerInterface = (*Handler)(nil)

// --- helpers ---

func errBody(msg string) map[string]string {
	return map[string]string{"error": msg}
}

// isNotFound определяет отсутствие записи в БД (pgx.ErrNoRows),
// в т.ч. завёрнутую через fmt.Errorf("...: %w").
func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isForbidden(err error) bool {
	return err != nil && strings.Contains(err.Error(), "forbidden")
}

func isInvalid(err error) bool {
	return err != nil && strings.Contains(err.Error(), "invalid status")
}

// isOrderProductError определяет ошибки создания заказа, связанные
// с отсутствием или недоступностью товара (маппятся на 400).
func isOrderProductError(err error) bool {
	return errors.Is(err, model.ErrProductNotFound) || errors.Is(err, model.ErrProductUnavailable)
}

// --- маппинг model -> api ---

func toAPIRestaurant(r model.Restaurant) api.Restaurant {
	return api.Restaurant{
		Id:          &r.ID,
		Name:        &r.Name,
		Description: &r.Description,
		Address:     &r.Address,
		Phone:       &r.Phone,
		IsActive:    &r.IsActive,
		CreatedAt:   &r.CreatedAt,
	}
}

func toAPIProduct(p model.Product) api.Product {
	return api.Product{
		Id:           &p.ID,
		RestaurantId: &p.RestaurantID,
		Name:         &p.Name,
		Description:  &p.Description,
		Price:        &p.Price,
		IsAvailable:  &p.IsAvailable,
	}
}

func toAPIOrderItem(i model.OrderItem) api.OrderItem {
	return api.OrderItem{
		Id:            &i.ID,
		SubOrderId:    &i.SubOrderID,
		ProductId:     &i.ProductID,
		Quantity:      &i.Quantity,
		PriceAtMoment: func() *float32 { v := float32(i.PriceAtMoment); return &v }(),
	}
}

func toAPISubOrder(s model.SubOrder) api.SubOrder {
	status := api.SubOrderStatus(s.Status)
	items := make([]api.OrderItem, len(s.Items))
	for i := range s.Items {
		items[i] = toAPIOrderItem(s.Items[i])
	}
	subtotal := float32(s.Subtotal)
	return api.SubOrder{
		Id:           &s.ID,
		OrderId:      &s.OrderID,
		RestaurantId: &s.RestaurantID,
		Status:       &status,
		Subtotal:     &subtotal,
		CreatedAt:    &s.CreatedAt,
		UpdatedAt:    &s.UpdatedAt,
		Items:        &items,
	}
}

func toAPIOrder(o model.Order) api.Order {
	subOrders := make([]api.SubOrder, len(o.SubOrders))
	for i := range o.SubOrders {
		subOrders[i] = toAPISubOrder(o.SubOrders[i])
	}
	total := float32(o.TotalPrice)
	return api.Order{
		Id:              &o.ID,
		ClientId:        &o.ClientID,
		DeliveryAddress: &o.DeliveryAddress,
		TotalPrice:      &total,
		CreatedAt:       &o.CreatedAt,
		SubOrders:       &subOrders,
	}
}

// --- пользовательские эндпоинты ---

// GetRestaurants возвращает список активных ресторанов.
// (GET /api/v1/restaurants)
func (h *Handler) GetRestaurants(ctx echo.Context) error {
	restaurants, err := h.restaurantUC.GetAll(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}

	out := make([]api.Restaurant, len(restaurants))
	for i := range restaurants {
		out[i] = toAPIRestaurant(restaurants[i])
	}
	return ctx.JSON(http.StatusOK, out)
}

// GetRestaurant возвращает детали ресторана.
// (GET /api/v1/restaurants/{id})
func (h *Handler) GetRestaurant(ctx echo.Context, id int) error {
	restaurant, err := h.restaurantUC.GetByID(ctx.Request().Context(), id)
	if err != nil {
		if isNotFound(err) {
			return ctx.JSON(http.StatusNotFound, errBody("restaurant not found"))
		}
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}
	return ctx.JSON(http.StatusOK, toAPIRestaurant(restaurant))
}

// GetRestaurantMenu возвращает меню ресторана.
// (GET /api/v1/restaurants/{id}/products)
func (h *Handler) GetRestaurantMenu(ctx echo.Context, id int) error {
	reqCtx := ctx.Request().Context()

	if _, err := h.restaurantUC.GetByID(reqCtx, id); err != nil {
		if isNotFound(err) {
			return ctx.JSON(http.StatusNotFound, errBody("restaurant not found"))
		}
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}

	products, err := h.restaurantUC.GetMenu(reqCtx, id)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}

	out := make([]api.Product, len(products))
	for i := range products {
		out[i] = toAPIProduct(products[i])
	}
	return ctx.JSON(http.StatusOK, out)
}

// CreateOrder создаёт заказ с группировкой позиций по ресторанам.
// (POST /api/v1/orders)
func (h *Handler) CreateOrder(ctx echo.Context) error {
	var req api.CreateOrderRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errBody("invalid request body"))
	}

	if req.DeliveryAddress == "" {
		return ctx.JSON(http.StatusBadRequest, errBody("delivery_address is required"))
	}
	if len(req.Items) == 0 {
		return ctx.JSON(http.StatusBadRequest, errBody("items must not be empty"))
	}

	items := make([]dto.OrderItem, len(req.Items))
	for i := range req.Items {
		if req.Items[i].ProductId == nil || req.Items[i].Quantity == nil || *req.Items[i].Quantity <= 0 {
			return ctx.JSON(http.StatusBadRequest, errBody("each item requires product_id and positive quantity"))
		}
		items[i] = dto.OrderItem{
			ProductID: *req.Items[i].ProductId,
			Quantity:  *req.Items[i].Quantity,
		}
	}

	order, err := h.orderUC.CreateOrder(ctx.Request().Context(), req.ClientId, req.DeliveryAddress, items)
	if err != nil {
		if isOrderProductError(err) {
			return ctx.JSON(http.StatusBadRequest, errBody(err.Error()))
		}
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}
	return ctx.JSON(http.StatusCreated, toAPIOrder(order))
}

// GetOrder возвращает заказ со всеми подзаказами.
// (GET /api/v1/orders/{id})
func (h *Handler) GetOrder(ctx echo.Context, id int) error {
	order, err := h.orderUC.GetOrderWithDetails(ctx.Request().Context(), id)
	if err != nil {
		if isNotFound(err) {
			return ctx.JSON(http.StatusNotFound, errBody("order not found"))
		}
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}
	return ctx.JSON(http.StatusOK, toAPIOrder(order))
}

// --- партнёрские эндпоинты (требуют RestaurantAuthMiddleware) ---

// restaurantIDOr401 извлекает restaurant_id из контекста, при отсутствии — 401.
func restaurantIDOr401(ctx echo.Context) (int, error) {
	restaurantID := GetRestaurantID(ctx)
	if restaurantID == 0 {
		return 0, ctx.JSON(http.StatusUnauthorized, errBody("unauthorized"))
	}
	return restaurantID, nil
}

// GetRestaurantOrders возвращает новые (pending) подзаказы ресторана.
// (GET /api/v1/restaurant/orders)
func (h *Handler) GetRestaurantOrders(ctx echo.Context) error {
	restaurantID, errResp := restaurantIDOr401(ctx)
	if errResp != nil {
		return errResp
	}

	subOrders, err := h.subOrderUC.GetPendingOrders(ctx.Request().Context(), restaurantID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}

	out := make([]api.SubOrder, len(subOrders))
	for i := range subOrders {
		out[i] = toAPISubOrder(subOrders[i])
	}
	return ctx.JSON(http.StatusOK, out)
}

// UpdateSubOrderStatus обновляет статус подзаказа.
// (PATCH /api/v1/restaurant/sub-orders/{id}/status)
func (h *Handler) UpdateSubOrderStatus(ctx echo.Context, id int) error {
	restaurantID, errResp := restaurantIDOr401(ctx)
	if errResp != nil {
		return errResp
	}

	var req api.UpdateSubOrderStatusRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errBody("invalid request body"))
	}

	updated, err := h.subOrderUC.UpdateStatus(
		ctx.Request().Context(),
		id,
		restaurantID,
		model.SubOrderStatus(req.Status),
	)
	if err != nil {
		switch {
		case isInvalid(err):
			return ctx.JSON(http.StatusBadRequest, errBody(err.Error()))
		case isNotFound(err) || isForbidden(err):
			return ctx.JSON(http.StatusNotFound, errBody("sub order not found"))
		default:
			return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
		}
	}
	return ctx.JSON(http.StatusOK, toAPISubOrder(updated))
}

// GetMyProducts возвращает продукты ресторана.
// (GET /api/v1/restaurant/products)
func (h *Handler) GetMyProducts(ctx echo.Context) error {
	restaurantID, errResp := restaurantIDOr401(ctx)
	if errResp != nil {
		return errResp
	}

	products, err := h.productUC.GetByRestaurant(ctx.Request().Context(), restaurantID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}

	out := make([]api.Product, len(products))
	for i := range products {
		out[i] = toAPIProduct(products[i])
	}
	return ctx.JSON(http.StatusOK, out)
}

// CreateProduct создаёт продукт с принудительной привязкой к ресторану.
// (POST /api/v1/restaurant/products)
func (h *Handler) CreateProduct(ctx echo.Context) error {
	restaurantID, errResp := restaurantIDOr401(ctx)
	if errResp != nil {
		return errResp
	}

	var req api.Product
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errBody("invalid request body"))
	}
	if req.Name == nil || *req.Name == "" {
		return ctx.JSON(http.StatusBadRequest, errBody("name is required"))
	}

	product := apiProductToModel(req, 0, restaurantID)
	created, err := h.productUC.CreateProduct(ctx.Request().Context(), product, restaurantID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}
	return ctx.JSON(http.StatusCreated, toAPIProduct(created))
}

// UpdateProduct обновляет продукт с проверкой владения.
// (PATCH /api/v1/restaurant/products/{id})
func (h *Handler) UpdateProduct(ctx echo.Context, id int) error {
	restaurantID, errResp := restaurantIDOr401(ctx)
	if errResp != nil {
		return errResp
	}

	var req api.Product
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, errBody("invalid request body"))
	}

	product := apiProductToModel(req, id, restaurantID)
	updated, err := h.productUC.UpdateProduct(ctx.Request().Context(), product, restaurantID)
	if err != nil {
		switch {
		case isNotFound(err) || isForbidden(err):
			return ctx.JSON(http.StatusNotFound, errBody("product not found"))
		default:
			return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
		}
	}
	return ctx.JSON(http.StatusOK, toAPIProduct(updated))
}

// DeleteProduct удаляет продукт с проверкой владения.
// (DELETE /api/v1/restaurant/products/{id})
func (h *Handler) DeleteProduct(ctx echo.Context, id int) error {
	restaurantID, errResp := restaurantIDOr401(ctx)
	if errResp != nil {
		return errResp
	}

	if err := h.productUC.DeleteProduct(ctx.Request().Context(), id, restaurantID); err != nil {
		if isNotFound(err) || isForbidden(err) {
			return ctx.JSON(http.StatusNotFound, errBody("product not found"))
		}
		return ctx.JSON(http.StatusInternalServerError, errBody(err.Error()))
	}
	return ctx.NoContent(http.StatusNoContent)
}

// apiProductToModel конвертирует api.Product в model.Product.
// ID берётся из пути, restaurant_id принудительно — из контекста.
func apiProductToModel(p api.Product, id, restaurantID int) model.Product {
	var out model.Product
	out.ID = id
	out.RestaurantID = restaurantID
	if p.Name != nil {
		out.Name = *p.Name
	}
	if p.Description != nil {
		out.Description = *p.Description
	}
	if p.Price != nil {
		out.Price = *p.Price
	}
	if p.IsAvailable != nil {
		out.IsAvailable = *p.IsAvailable
	}
	return out
}
