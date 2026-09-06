package delivery

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/slipynil/avito-kitchen/internal/dto"
	"github.com/slipynil/avito-kitchen/internal/usecase"
)

type Handler struct {
	productUC usecase.ProductUseCase
	orderUC   usecase.OrderUseCase
}

func NewHandler(productUC usecase.ProductUseCase, orderUC usecase.OrderUseCase) *Handler {
	return &Handler{
		productUC: productUC,
		orderUC:   orderUC,
	}
}

// Регистрация маршрутов
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/products", h.GetProducts)
	g.POST("/orders", h.CreateOrder)
	g.GET("/orders/:id", h.GetOrderStatus)
}

// Хендлер для получения списка продуктов
// клиент - отправляет пустой GET запрос
// сервер - возвращает список продуктов
func (h *Handler) GetProducts(c *echo.Context) error {
	products, err := h.productUC.GetProducts(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, products)
}

// Хендлер для создания заказа
// клиент - отправляет слайс с данными о товарах
// сервер - возвращает созданный заказ
func (h *Handler) CreateOrder(c *echo.Context) error {
	var req dto.CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	order, err := h.orderUC.CreateOrder(c.Request().Context(), req.ClientID, req.DeliveryAddress, req.Items)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, order)
}

// Хендлер для получения статуса заказа
// клиент - отправляет GET запрос с id заказа
// сервер - возвращает статус заказа
func (h *Handler) GetOrderStatus(c *echo.Context) error {
	orderID := c.Param("id")
	userID := c.Request().Header.Get("X-User-ID")

	status, err := h.orderUC.GetOrderStatus(c.Request().Context(), orderID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, status)
}
