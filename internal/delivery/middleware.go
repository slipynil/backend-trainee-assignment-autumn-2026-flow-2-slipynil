package delivery

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/slipynil/avito-kitchen/internal/usecase"
)

// RestaurantIDKey ключ, под которым restaurant_id хранится в echo-контексте.
const RestaurantIDKey = "restaurant_id"

// RestaurantAuthMiddleware проверяет заголовок X-Restaurant-API-Key через
// restaurantUC.ValidateAPIKey и кладёт restaurant_id в контекст.
// При отсутствии или невалидном ключе возвращает 401.
func RestaurantAuthMiddleware(restaurantUC *usecase.RestaurantUseCase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := c.Request().Header.Get("X-Restaurant-API-Key")
			if apiKey == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing X-Restaurant-API-Key header"})
			}

			restaurantID, err := restaurantUC.ValidateAPIKey(c.Request().Context(), apiKey)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
			}

			c.Set(RestaurantIDKey, restaurantID)
			return next(c)
		}
	}
}

// GetRestaurantID возвращает restaurant_id, положенный middleware в контекст.
// Возвращает 0, если значение отсутствует или имеет неожиданный тип.
func GetRestaurantID(c echo.Context) int {
	switch v := c.Get(RestaurantIDKey).(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
