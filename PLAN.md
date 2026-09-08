# PLAN.md — Детальный план реализации Авито.Кухня

## Принятые решения

1. **Заказы из нескольких ресторанов** — SubOrders: каждый ресторан имеет свой статус приготовления
2. **Авторизация ресторанов** — API key в заголовке `X-Restaurant-API-Key`
3. **Кодогенерация** — oapi-codegen для сервера и клиента
4. **Язык restaurant-service** — Golang (такой же стек)
5. **Получение заказов** — polling (опрос API каждые 5 секунд)
6. **Категории товаров** — не нужны в MVP

---

## Схема БД

### Текущие таблицы

- `users` — пользователи платформы
- `products` — товары (привязаны к ресторанам)
- `orders` — общие заказы пользователей
- `order_items` — позиции подзаказов

### Новые таблицы

- `restaurants` — заведения на платформе
- `restaurant_api_keys` — API ключи для авторизации ресторанов
- `sub_orders` — подзаказы для каждого ресторана в рамках общего заказа

### ER-диаграмма

```
users 1---* orders
orders 1---* sub_orders
restaurants 1---* sub_orders
restaurants 1---* products
restaurants 1---* restaurant_api_keys
sub_orders 1---* order_items
products 1---* order_items
```

---

## Этап 1: Подготовка инфраструктуры

### 1.1 Создать `.golangci.yml`

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - gosimple
    - ineffassign
    - unused
    - misspell

issues:
  exclude-use-default: false
```

### 1.2 Расширить `mise.toml`

`Makefile` не создаем — все команды живут в `mise.toml`.
Сохранить существующее (`[tools] go`, `[env]`, `server/client/migrate-up/migrate-down`)
и добавить задачи:

```toml
[tasks.lint]
description = "Линтинг"
run = "golangci-lint run ./..."

[tasks.generate]
description = "Генерация API из OpenAPI"
run = "oapi-codegen -package api -generate types,server,client -o internal/api/generated.go docs/openapi.yaml"

[tasks.build]
description = "Сборка обоих сервисов"
run = "go build -o bin/server ./cmd/server && go build -o bin/restaurant-service ./cmd/restaurant-service"

[tasks.up]
description = "Поднять всё в Docker"
run = "docker compose up -d --build"

[tasks.down]
description = "Остановить всё"
run = "docker compose down"

[tasks.logs]
description = "Логи всех сервисов"
run = "docker compose logs -f"
```

Правило: везде использовать `docker compose` (без дефиса), не `docker-compose`.

### 1.3 Установить oapi-codegen

```bash
go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest
```

---

## Этап 2: OpenAPI спецификация

Создать `docs/openapi.yaml` с описанием всех эндпоинтов:

### Пользовательское API (`/api/v1/`)

- `GET /restaurants` — список ресторанов
- `GET /restaurants/{id}` — детали ресторана
- `GET /restaurants/{id}/products` — меню ресторана
- `POST /orders` — создание заказа (товары из разных ресторанов)
- `GET /orders/{id}` — детали заказа с подзаказами

### Партнерское API (`/api/v1/restaurant/`) — требует заголовок `X-Restaurant-API-Key`

- `GET /restaurant/orders` — получить новые подзаказы (polling)
- `PATCH /restaurant/sub-orders/{id}/status` — обновить статус подзаказа
- `GET /restaurant/products` — список своих продуктов
- `POST /restaurant/products` — добавить продукт
- `PATCH /restaurant/products/{id}` — обновить продукт (цена, наличие)
- `DELETE /restaurant/products/{id}` — удалить продукт

### Схемы данных

```yaml
components:
  schemas:
    Restaurant:
      type: object
      properties:
        id: {type: integer}
        name: {type: string}
        description: {type: string}
        address: {type: string}
        phone: {type: string}
        is_active: {type: boolean}

    Product:
      type: object
      properties:
        id: {type: integer}
        restaurant_id: {type: integer}
        name: {type: string}
        description: {type: string}
        price: {type: number, format: double}
        is_available: {type: boolean}

    Order:
      type: object
      properties:
        id: {type: integer}
        client_id: {type: integer}
        delivery_address: {type: string}
        total_price: {type: number}
        created_at: {type: string, format: date-time}
        sub_orders:
          type: array
          items:
            $ref: '#/components/schemas/SubOrder'

    SubOrder:
      type: object
      properties:
        id: {type: integer}
        order_id: {type: integer}
        restaurant_id: {type: integer}
        status:
          type: string
          enum: [pending, accepted, preparing, ready, cancelled]
        subtotal: {type: number}
        items:
          type: array
          items:
            $ref: '#/components/schemas/OrderItem'

    OrderItem:
      type: object
      properties:
        id: {type: integer}
        sub_order_id: {type: integer}
        product_id: {type: integer}
        quantity: {type: integer}
        price_at_moment: {type: number}

    CreateOrderRequest:
      type: object
      required: [client_id, delivery_address, items]
      properties:
        client_id: {type: integer}
        delivery_address: {type: string}
        items:
          type: array
          items:
            type: object
            properties:
              product_id: {type: integer}
              quantity: {type: integer}

    UpdateSubOrderStatusRequest:
      type: object
      required: [status]
      properties:
        status:
          type: string
          enum: [accepted, preparing, ready, cancelled]

  securitySchemes:
    RestaurantApiKey:
      type: apiKey
      in: header
      name: X-Restaurant-API-Key
```

---

## Этап 3: Миграции БД

### 3.1 `migrations/000002_add_restaurants.up.sql`

```sql
-- Таблица ресторанов
CREATE TABLE restaurants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    address VARCHAR(500) NOT NULL,
    phone VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- API ключи ресторанов
CREATE TABLE restaurant_api_keys (
    id SERIAL PRIMARY KEY,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    api_key VARCHAR(64) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Обновляем products
ALTER TABLE products ADD COLUMN restaurant_id INTEGER;
ALTER TABLE products ADD COLUMN description TEXT;
ALTER TABLE products ADD COLUMN is_available BOOLEAN NOT NULL DEFAULT true;

-- Подзаказы
CREATE TABLE sub_orders (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    subtotal DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Новая таблица order_items с привязкой к sub_orders
CREATE TABLE order_items_new (
    id SERIAL PRIMARY KEY,
    sub_order_id INTEGER NOT NULL REFERENCES sub_orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    price_at_moment DECIMAL(10, 2) NOT NULL
);

DROP TABLE IF EXISTS order_items;
ALTER TABLE order_items_new RENAME TO order_items;

-- Индексы
CREATE INDEX idx_products_restaurant_id ON products(restaurant_id);
CREATE INDEX idx_sub_orders_order_id ON sub_orders(order_id);
CREATE INDEX idx_sub_orders_restaurant_id ON sub_orders(restaurant_id);
CREATE INDEX idx_sub_orders_status ON sub_orders(status);
CREATE INDEX idx_order_items_sub_order_id ON order_items(sub_order_id);
CREATE INDEX idx_restaurant_api_keys_api_key ON restaurant_api_keys(api_key);
```

### 3.2 `migrations/000002_add_restaurants.down.sql`

```sql
DROP INDEX IF EXISTS idx_restaurant_api_keys_api_key;
DROP INDEX IF EXISTS idx_order_items_sub_order_id;
DROP INDEX IF EXISTS idx_sub_orders_status;
DROP INDEX IF EXISTS idx_sub_orders_restaurant_id;
DROP INDEX IF EXISTS idx_sub_orders_order_id;
DROP INDEX IF EXISTS idx_products_restaurant_id;

DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS sub_orders;
DROP TABLE IF EXISTS restaurant_api_keys;

ALTER TABLE products DROP COLUMN IF EXISTS is_available;
ALTER TABLE products DROP COLUMN IF EXISTS description;
ALTER TABLE products DROP COLUMN IF EXISTS restaurant_id;

DROP TABLE IF EXISTS restaurants;

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    price_at_moment DECIMAL(10, 2) NOT NULL
);
```

### 3.3 `migrations/000003_seed_data.up.sql`

```sql
-- Рестораны
INSERT INTO restaurants (name, description, address, phone) VALUES
('Додо Пицца', 'Сеть пиццерий с доставкой', 'ул. Ленина, 1', '+7-900-123-45-01'),
('Тануки', 'Японская кухня и суши', 'ул. Пушкина, 10', '+7-900-123-45-02'),
('Burger King', 'Бургеры и фастфуд', 'пр. Мира, 5', '+7-900-123-45-03');

-- API ключи
INSERT INTO restaurant_api_keys (restaurant_id, api_key) VALUES
(1, 'dodo-pizza-secret-key-12345'),
(2, 'tanuki-secret-key-67890'),
(3, 'burger-king-secret-key-11111');

-- Продукты Додо Пиццы
INSERT INTO products (restaurant_id, name, description, price, is_available) VALUES
(1, 'Маргарита', 'Классическая пицца с томатами и моцареллой', 450.00, true),
(1, 'Пепперони', 'Острая пицца с колбасой пепперони', 550.00, true),
(1, '4 сыра', 'Пицца с четырьмя видами сыра', 600.00, true),
(1, 'Coca-Cola 0.5л', 'Газированный напиток', 100.00, true),
(1, 'Картофель фри', 'Хрустящий картофель', 150.00, true);

-- Продукты Тануки
INSERT INTO products (restaurant_id, name, description, price, is_available) VALUES
(2, 'Филадельфия', 'Ролл с лососем и сливочным сыром', 350.00, true),
(2, 'Калифорния', 'Ролл с крабом и авокадо', 300.00, true),
(2, 'Унаги маки', 'Ролл с угрем', 280.00, true),
(2, 'Мисо-суп', 'Традиционный японский суп', 150.00, true),
(2, 'Эдамаме', 'Молодые соевые бобы', 200.00, true);

-- Продукты Burger King
INSERT INTO products (restaurant_id, name, description, price, is_available) VALUES
(3, 'Воппер', 'Легендарный бургер с говядиной', 280.00, true),
(3, 'Чикен бургер', 'Бургер с курицей', 250.00, true),
(3, 'Картофель фри большой', 'Хрустящий картофель 150г', 120.00, true),
(3, 'Наггетсы 6 шт', 'Куриные наггетсы', 180.00, true),
(3, 'Milkshake ванильный', 'Молочный коктейль', 150.00, true);

-- Тестовые пользователи
INSERT INTO users (role) VALUES ('customer'), ('customer'), ('customer');

-- Делаем restaurant_id обязательным
ALTER TABLE products ALTER COLUMN restaurant_id SET NOT NULL;
ALTER TABLE products ADD CONSTRAINT products_restaurant_id_fkey
    FOREIGN KEY (restaurant_id) REFERENCES restaurants(id) ON DELETE CASCADE;
```

### 3.4 `migrations/000003_seed_data.down.sql`

```sql
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_restaurant_id_fkey;
TRUNCATE TABLE order_items, sub_orders, orders, products, restaurant_api_keys, restaurants, users RESTART IDENTITY CASCADE;
```

---

## Этап 4: Обновление моделей

### 4.1 `internal/model/models.go`

Добавить новые модели и обновить существующие:

```go
// Restaurant заведение на платформе
type Restaurant struct {
    ID          int       `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Description string    `json:"description" db:"description"`
    Address     string    `json:"address" db:"address"`
    Phone       string    `json:"phone" db:"phone"`
    IsActive    bool      `json:"is_active" db:"is_active"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RestaurantAPIKey API ключ ресторана
type RestaurantAPIKey struct {
    ID           int       `json:"id" db:"id"`
    RestaurantID int       `json:"restaurant_id" db:"restaurant_id"`
    APIKey       string    `json:"api_key" db:"api_key"`
    IsActive     bool      `json:"is_active" db:"is_active"`
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// SubOrder подзаказ для конкретного ресторана
type SubOrder struct {
    ID           int             `json:"id" db:"id"`
    OrderID      int             `json:"order_id" db:"order_id"`
    RestaurantID int             `json:"restaurant_id" db:"restaurant_id"`
    Status       SubOrderStatus  `json:"status" db:"status"`
    Subtotal     float64         `json:"subtotal" db:"subtotal"`
    CreatedAt    time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
    Items        []OrderItem     `json:"items,omitempty"`
}

// SubOrderStatus статусы подзаказа
type SubOrderStatus string

const (
    SubOrderStatusPending   SubOrderStatus = "pending"
    SubOrderStatusAccepted  SubOrderStatus = "accepted"
    SubOrderStatusPreparing SubOrderStatus = "preparing"
    SubOrderStatusReady     SubOrderStatus = "ready"
    SubOrderStatusCancelled SubOrderStatus = "cancelled"
)
```

Обновить `Product`:

```go
type Product struct {
    ID           int     `json:"id" db:"id"`
    RestaurantID int     `json:"restaurant_id" db:"restaurant_id"`
    Name         string  `json:"name" db:"name"`
    Description  string  `json:"description" db:"description"`
    Price        float64 `json:"price" db:"price"`
    IsAvailable  bool    `json:"is_available" db:"is_available"`
}
```

Обновить `OrderItem`:

```go
type OrderItem struct {
    ID            int     `json:"id" db:"id"`
    SubOrderID    int     `json:"sub_order_id" db:"sub_order_id"`
    ProductID     int     `json:"product_id" db:"product_id"`
    Quantity      int     `json:"quantity" db:"quantity"`
    PriceAtMoment float64 `json:"price_at_moment" db:"price_at_moment"`
}
```

Обновить `Order`:

```go
type Order struct {
    ID              int         `json:"id" db:"id"`
    ClientID        int         `json:"client_id" db:"client_id"`
    Status          OrderStatus `json:"status" db:"status"`
    TotalPrice      float64     `json:"total_price" db:"total_price"`
    CreatedAt       time.Time   `json:"created_at" db:"created_at"`
    DeliveryAddress string      `json:"delivery_address" db:"delivery_address"`
    SubOrders       []SubOrder  `json:"sub_orders,omitempty"`
}
```

---

## Этап 5: Repository слой

### 5.1 Создать `internal/repository/restaurant.go`

Методы:
- `GetAll(ctx)` — все активные рестораны
- `GetByID(ctx, id)` — ресторан по ID
- `GetByAPIKey(ctx, apiKey)` — restaurant_id по API ключу

### 5.2 Создать `internal/repository/sub_order.go`

Методы:
- `Create(ctx, orderID, restaurantID, subtotal)` — создание подзаказа
- `GetByOrderID(ctx, orderID)` — подзаказы заказа
- `GetByRestaurant(ctx, restaurantID, status)` — подзаказы ресторана с фильтром по статусу
- `UpdateStatus(ctx, subOrderID, status)` — обновление статуса
- `GetByID(ctx, id)` — подзаказ по ID

### 5.3 Обновить `internal/repository/order.go`

Методы:
- `Create(ctx, clientID, deliveryAddress, items)` — создание заказа с группировкой items по ресторанам и созданием подзаказов
- `GetByID(ctx, id)` — заказ по ID
- `GetByIDWithSubOrders(ctx, orderID)` — заказ со всеми подзаказами и items

Логика `Create`:
1. Запросить `restaurant_id` и `price` для каждого `product_id`
2. Сгруппировать items по `restaurant_id`
3. Создать `order`
4. Для каждой группы создать `sub_order`
5. Создать `order_items` с привязкой к `sub_order`

### 5.4 Обновить `internal/repository/product.go`

Добавить методы:
- `GetByRestaurant(ctx, restaurantID)` — продукты ресторана
- `Create(ctx, product)` — создание продукта
- `Update(ctx, product)` — обновление продукта
- `Delete(ctx, id)` — удаление продукта

---

## Этап 6: UseCase слой

### 6.1 Создать `internal/usecase/restaurant.go`

Методы:
- `GetAll(ctx)` — все рестораны
- `GetByID(ctx, id)` — ресторан по ID
- `GetMenu(ctx, restaurantID)` — меню ресторана
- `ValidateAPIKey(ctx, apiKey)` — проверка API ключа

### 6.2 Создать `internal/usecase/sub_order.go`

Методы:
- `GetPendingOrders(ctx, restaurantID)` — новые подзаказы для ресторана
- `UpdateStatus(ctx, subOrderID, restaurantID, status)` — обновление статуса (с проверкой владения)

### 6.3 Обновить `internal/usecase/order.go`

- `CreateOrder(ctx, clientID, deliveryAddress, items)` — создание заказа
- `GetOrderWithDetails(ctx, orderID)` — заказ со всеми подзаказами

### 6.4 Обновить `internal/usecase/product.go`

- `GetByRestaurant(ctx, restaurantID)` — продукты ресторана
- `CreateProduct(ctx, product, restaurantID)` — создание продукта (с принудительной привязкой к ресторану)
- `UpdateProduct(ctx, product, restaurantID)` — обновление (с проверкой владения)
- `DeleteProduct(ctx, productID, restaurantID)` — удаление (с проверкой владения)

---

## Этап 7: Кодогенерация API

### 7.1 Запустить генерацию

```bash
oapi-codegen -package api -generate types,server,client -o internal/api/generated.go docs/openapi.yaml
```

Это создаст:
- Типы (structs) для всех схем
- Интерфейс `ServerInterface` с методами для всех эндпоинтов
- Клиента для вызова API

---

## Этап 8: Handler и Middleware

### 8.1 Создать `internal/delivery/middleware.go`

`RestaurantAuthMiddleware`:
- Извлекает заголовок `X-Restaurant-API-Key`
- Проверяет ключ через `restaurantUC.ValidateAPIKey`
- Добавляет `restaurant_id` в контекст

### 8.2 Обновить `internal/delivery/handler.go`

Реализовать интерфейс `ServerInterface` из сгенерированного кода.

**Пользовательские эндпоинты:**
- `GetRestaurants` — список ресторанов
- `GetRestaurant` — детали ресторана
- `GetRestaurantMenu` — меню ресторана
- `CreateOrder` — создание заказа
- `GetOrder` — детали заказа с подзаказами

**Партнерские эндпоинты** (с middleware):
- `GetRestaurantOrders` — новые подзаказы ресторана
- `UpdateSubOrderStatus` — обновление статуса подзаказа
- `GetMyProducts` — список продуктов ресторана
- `CreateProduct` — добавление продукта
- `UpdateProduct` — обновление продукта
- `DeleteProduct` — удаление продукта

---

## Этап 9: Обновить main.go

### 9.1 `cmd/server/main.go`

```go
func main() {
    // Подключение к БД
    // Инициализация репозиториев: productRepo, orderRepo, restaurantRepo, subOrderRepo
    // Инициализация use cases: productUC, orderUC, restaurantUC, subOrderUC
    // Инициализация handler: NewHandler(productUC, orderUC, restaurantUC, subOrderUC)
    // Настройка Echo, регистрация маршрутов
    // Запуск сервера на порту 8080
    // Graceful shutdown
}
```

---

## Этап 10: Restaurant Service

### 10.1 `cmd/restaurant-service/main.go`

Микросервис, имитирующий работу реального ресторана.

**Логика:**
1. Polling loop каждые 5 секунд
2. Запрос `GET /api/v1/restaurant/orders` с API ключом
3. Для каждого нового подзаказа запускается горутина обработки:
   - `pending` → `accepted` (через 3 сек)
   - `accepted` → `preparing` (через 5 сек)
   - `preparing` → `ready` (через 10 сек)
4. Каждый статус обновляется через `PATCH /api/v1/restaurant/sub-orders/{id}/status`

**Переменные окружения:**
- `PLATFORM_API_URL` — URL основного сервиса
- `RESTAURANT_API_KEY` — API ключ ресторана
- `RESTAURANT_ID` — ID ресторана

### 10.2 `cmd/restaurant-service/Dockerfile`

Многоэтапная сборка (builder → alpine).

---

## Этап 11: Docker Compose

### 11.1 Обновить `docker-compose.yml`

Добавить сервис `restaurant-service`:

```yaml
restaurant-service:
  restart: always
  build:
    context: .
    dockerfile: cmd/restaurant-service/Dockerfile
  environment:
    - PLATFORM_API_URL=http://service:8080
    - RESTAURANT_API_KEY=dodo-pizza-secret-key-12345
    - RESTAURANT_ID=1
  depends_on:
    - service
```

Обновить сервис `service`:

```yaml
service:
  ports:
    - "8080:8080"
  environment:
    - PORT=8080
```

---

## Этап 12: Документация

### 12.1 CJM диаграммы (PlantUML)

**`docs/cjm-user.puml`** — Customer Journey Map пользователя:
1. Просмотр списка ресторанов
2. Просмотр меню ресторана
3. Выбор товаров из разных ресторанов
4. Оформление заказа (с общим адресом доставки)
5. Получение заказа
6. Точки трения: товар недоступен, ресторан отменил, разное время приготовления

**`docs/cjm-restaurant.puml`** — CJM ресторана:
1. Регистрация на платформе (получение API ключа)
2. Загрузка меню
3. Получение новых заказов (polling)
4. Принятие заказа
5. Приготовление
6. Готовность к выдаче
7. Управление наличием товаров

### 12.2 C4 диаграммы

**`docs/c4-container.puml`** (Level 2):
- Пользователь → API Service → PostgreSQL
- Restaurant Service → API Service

**`docs/c4-component.puml`** (Level 3):
- HTTP Handlers → Middleware → Use Cases → Repositories → Database

### 12.3 ER-диаграмма

**`docs/er-diagram.puml`** — все таблицы и связи

### 12.4 README.md

Структура:
1. Описание проекта
2. Архитектура (C4 диаграммы)
3. Схема БД (ER-диаграмма)
4. API эндпоинты (ссылка на OpenAPI)
5. Установка и запуск
6. Примеры запросов
7. CJM диаграммы
8. Допущения и упрощения MVP
9. Планы на масштабирование
10. AI-first подход (промпт)

---

## Чеклист выполнения

### Миграции
- [ ] `000002_add_restaurants.up.sql`
- [ ] `000002_add_restaurants.down.sql`
- [ ] `000003_seed_data.up.sql`
- [ ] `000003_seed_data.down.sql`

### Модели
- [ ] `Restaurant`, `RestaurantAPIKey`, `SubOrder` добавлены
- [ ] `Product` обновлен (restaurant_id, description, is_available)
- [ ] `OrderItem` обновлен (sub_order_id вместо order_id)
- [ ] `Order` обновлен (SubOrders []SubOrder)

### Repository
- [ ] `restaurant.go` создан
- [ ] `sub_order.go` создан
- [ ] `order.go` обновлен (группировка по ресторанам)
- [ ] `product.go` обновлен (CRUD для ресторана)

### UseCase
- [ ] `restaurant.go` создан
- [ ] `sub_order.go` создан
- [ ] `order.go` обновлен
- [ ] `product.go` обновлен

### API
- [ ] `docs/openapi.yaml` создан
- [ ] `internal/api/generated.go` сгенерирован
- [ ] `internal/delivery/middleware.go` создан
- [ ] `internal/delivery/handler.go` обновлен

### Инфраструктура
- [ ] `.golangci.yml` создан
- [ ] `mise.toml` расширен (lint/generate/build/up/down/logs)
- [ ] `cmd/server/main.go` обновлен
- [ ] `cmd/restaurant-service/main.go` создан
- [ ] `cmd/restaurant-service/Dockerfile` создан
- [ ] `docker-compose.yml` обновлен

### Документация
- [ ] `docs/cjm-user.puml` создан
- [ ] `docs/cjm-restaurant.puml` создан
- [ ] `docs/c4-container.puml` создан
- [ ] `docs/c4-component.puml` создан
- [ ] `docs/er-diagram.puml` создан
- [ ] `README.md` создан
- [ ] `PLAN.md` создан

### Проверка
- [ ] `go build ./...` — компиляция без ошибок
- [ ] `golangci-lint run ./...` — линтинг без ошибок
- [ ] `mise run up` (docker compose up -d --build) — всё запускается
- [ ] Restaurant service получает и обрабатывает заказы
