# Авито.Кухня — MVP сервиса заказа еды из нескольких ресторанов

## 1. Описание проекта

**Авито.Кухня** — backend-платформа доставки еды, расширяющая функционал Авито. Пользователь собирает **один заказ из товаров разных ресторанов** (например, пицца из «Додо Пиццы» + роллы из «Тануки»), указывает общий адрес доставки, а платформа автоматически разбивает заказ на **подзаказы (sub_orders)** — по одному на каждый ресторан, каждый со своим статусом приготовления.

Два контура API:

- **Пользовательское API** (`/api/v1/`) — без авторизации: каталог ресторанов, меню, создание и просмотр заказов.
- **Партнёрское API** (`/api/v1/restaurant/`) — для заведений, авторизация по заголовку `X-Restaurant-API-Key`: polling новых подзаказов, смена статусов, управление меню.

Отдельный сервис **`restaurant-service`** имитирует реальное заведение: опрашивает API каждые 5 секунд и прогоняет подзаказы по цепочке `pending → accepted → preparing → ready`.

Стек: **Go + Echo, PostgreSQL 18, Docker Compose, oapi-codegen, golangci-lint**.

## 2. Архитектура (C4)

### Level 2 — Containers (`docs/c4-container.puml`)

```plantuml
Пользователь -> API Service -> PostgreSQL
Restaurant Service -> API Service
```

- `API Service` — единый HTTP API (пользовательский + партнёрский контуры).
- `Restaurant Service` — имитатор заведения (polling + смена статусов).
- `PostgreSQL` — все таблицы платформы.
- `Migrate` — применяет SQL-миграции из `./migrations` при старте.

Исходник диаграммы: [`docs/c4-container.puml`](docs/c4-container.puml). Рендер: вставить содержимое в [plantuml.com](https://www.plantuml.com/plantuml/) или `plantuml docs/c4-container.puml`.

### Level 3 — Components (`docs/c4-component.puml`)

```plantuml
HTTP Handlers -> Middleware -> Use Cases -> Repositories -> Database
```

- `internal/delivery` — HTTP Handlers (реализуют `ServerInterface` из oapi-codegen) + `RestaurantAuthMiddleware` (проверка `X-Restaurant-API-Key`, `restaurant_id` в контекст).
- `internal/usecase` — `Restaurant` (каталог, `ValidateAPIKey`), `SubOrder` (polling, смена статуса с проверкой владения), `Order` (создание с группировкой позиций по ресторанам), `Product` (CRUD меню с проверкой владения).
- `internal/repository` — `restaurant.go`, `sub_order.go`, `order.go`, `product.go`.
- `internal/model` — `Restaurant`, `RestaurantAPIKey`, `SubOrder`/`SubOrderStatus`, обновлённые `Product`, `Order`, `OrderItem`.

Исходник: [`docs/c4-component.puml`](docs/c4-component.puml).

## 3. Схема БД (ER)

Исходник: [`docs/er-diagram.puml`](docs/er-diagram.puml). Миграции: `migrations/000001_init.up.sql`, `migrations/000002_add_restaurants.up.sql`, `migrations/000003_seed_data.up.sql`.

| Таблица | Назначение | Ключевые поля |
|---|---|---|
| `users` | Пользователи платформы | `id`, `role`, `created_at` |
| `restaurants` | Заведения | `id`, `name`, `address`, `phone`, `is_active` |
| `restaurant_api_keys` | API-ключи заведений | `restaurant_id → restaurants`, `api_key UNIQUE` |
| `products` | Товары меню | `restaurant_id → restaurants NOT NULL`, `price`, `is_available` |
| `orders` | Общие заказы | `client_id`, `status`, `total_price`, `delivery_address` |
| `sub_orders` | Подзаказы (по одному на ресторан) | `order_id → orders CASCADE`, `restaurant_id → restaurants`, `status` (`pending/accepted/preparing/ready/cancelled`), `subtotal` |
| `order_items` | Позиции подзаказов | `sub_order_id → sub_orders CASCADE`, `product_id → products`, `quantity`, `price_at_moment` |

Связи (по `PLAN.md`):

```text
users 1---* orders
orders 1---* sub_orders
restaurants 1---* sub_orders
restaurants 1---* products
restaurants 1---* restaurant_api_keys
sub_orders 1---* order_items
products 1---* order_items
```

Нюанс: `orders.client_id` хранится без FK на `users` (см. `000001_init.up.sql`) — историческое решение базовой схемы.

## 4. API

Полная спецификация: [`docs/openapi.yaml`](docs/openapi.yaml). Сервер и клиент генерируются: `mise run generate` (`oapi-codegen -package api -generate types,server,client -o internal/api/generated.go docs/openapi.yaml`).

### Пользовательское API (`/api/v1/`, без авторизации)

| # | Метод | Путь | Назначение |
|---|---|---|---|
| 1 | GET | `/api/v1/restaurants` | Список ресторанов |
| 2 | GET | `/api/v1/restaurants/{id}` | Детали ресторана |
| 3 | GET | `/api/v1/restaurants/{id}/products` | Меню ресторана |
| 4 | POST | `/api/v1/orders` | Создание заказа (товары из разных ресторанов) |
| 5 | GET | `/api/v1/orders/{id}` | Детали заказа с подзаказами |

### Партнёрское API (`/api/v1/restaurant/`, заголовок `X-Restaurant-API-Key`)

| # | Метод | Путь | Назначение |
|---|---|---|---|
| 6 | GET | `/api/v1/restaurant/orders` | Новые подзаказы ресторана (polling) |
| 7 | PATCH | `/api/v1/restaurant/sub-orders/{id}/status` | Обновить статус (`accepted/preparing/ready/cancelled`) |
| 8 | GET | `/api/v1/restaurant/products` | Список своих продуктов |
| 9 | POST | `/api/v1/restaurant/products` | Добавить продукт |
| 10 | PATCH | `/api/v1/restaurant/products/{id}` | Обновить продукт (цена, наличие) |
| 11 | DELETE | `/api/v1/restaurant/products/{id}` | Удалить продукт |

Всего **11 эндпоинтов**: 5 пользовательских + 6 партнёрских.

## 5. Установка и запуск

Требования: Docker + Docker Compose, Go 1.26.4+ (для локальной сборки), `oapi-codegen` и `golangci-lint` (опционально).

```bash
# 1. Клонировать и перейти в каталог
git clone <repo-url> && cd backend-trainee-assignment-autumn-2026-flow-2-slipynil-98560e3e

# 2. Переменные окружения (.env уже содержит дефолты)
cat .env
# DB_CONN=postgres://postgres:pass@localhost:5432/postgres?sslmode=disable
# DB_USER=postgres
# DB_NAME=postgres
# DB_PASSWD=pass

# 3. Поднять всё (postgres -> migrate -> service)
mise run up
# или: docker compose up -d --build

# 4. Проверить логи
mise run logs

# 5. API доступен на http://localhost:8080
curl http://localhost:8080/api/v1/restaurants
```

Миграции применяются автоматически контейнером `migrate` (`migrate/migrate -path /migrations -database ${DB_CONN} up`): `000001_init` (базовые таблицы) → `000002_add_restaurants` (рестораны, ключи, подзаказы) → `000003_seed_data` (Додо Пицца, Тануки, Burger King + товары + ключи + тестовые пользователи).

Полезное:

```bash
mise run build     # сборка bin/server и bin/restaurant-service
mise run generate  # перегенерация API из docs/openapi.yaml
mise run lint      # golangci-lint run ./...
mise run down      # остановить всё
```

## 6. Примеры запросов

Сиды: Додо Пицца (`id=1`, ключ `dodo-pizza-secret-key-12345`), Тануки (`id=2`, ключ `tanuki-secret-key-67890`), Burger King (`id=3`, ключ `burger-king-secret-key-11111`). Товары Додо: `1` Маргарита 450 ₽, `2` Пепперони 550 ₽; Тануки: `6` Филадельфия 350 ₽, `7` Калифорния 300 ₽.

### Список ресторанов

```bash
curl http://localhost:8080/api/v1/restaurants
# [{"id":1,"name":"Додо Пицца",...},{"id":2,"name":"Тануки",...},{"id":3,"name":"Burger King",...}]
```

### Меню ресторана

```bash
curl http://localhost:8080/api/v1/restaurants/1/products
curl http://localhost:8080/api/v1/restaurants/2/products
```

### Создание заказа из 2 ресторанов (Пепперони из Додо + 2× Филадельфия из Тануки)

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H 'Content-Type: application/json' \
  -d '{
    "client_id": 1,
    "delivery_address": "ул. Ленина, 1, кв. 10",
    "items": [
      {"product_id": 2, "quantity": 1},
      {"product_id": 6, "quantity": 2}
    ]
  }'
# 201: заказ total_price=1250.00, два sub_orders в статусе pending
```

### Детали заказа с подзаказами

```bash
curl http://localhost:8080/api/v1/orders/1
```

### Polling новых подзаказов (сторона ресторана, пример — Додо Пицца)

```bash
curl http://localhost:8080/api/v1/restaurant/orders \
  -H 'X-Restaurant-API-Key: dodo-pizza-secret-key-12345'
```

### Смена статуса подзаказа (принять → готовить → готов)

```bash
curl -X PATCH http://localhost:8080/api/v1/restaurant/sub-orders/1/status \
  -H 'Content-Type: application/json' \
  -H 'X-Restaurant-API-Key: dodo-pizza-secret-key-12345' \
  -d '{"status": "accepted"}'

curl -X PATCH http://localhost:8080/api/v1/restaurant/sub-orders/1/status \
  -H 'Content-Type: application/json' \
  -H 'X-Restaurant-API-Key: dodo-pizza-secret-key-12345' \
  -d '{"status": "preparing"}'

curl -X PATCH http://localhost:8080/api/v1/restaurant/sub-orders/1/status \
  -H 'Content-Type: application/json' \
  -H 'X-Restaurant-API-Key: dodo-pizza-secret-key-12345' \
  -d '{"status": "ready"}'
```

### Управление меню (снять товар с продажи)

```bash
curl -X PATCH http://localhost:8080/api/v1/restaurant/products/2 \
  -H 'Content-Type: application/json' \
  -H 'X-Restaurant-API-Key: dodo-pizza-secret-key-12345' \
  -d '{"is_available": false}'
```

## 7. CJM

| Диаграмма | Файл | Содержание |
|---|---|---|
| CJM пользователя | [`docs/cjm-user.puml`](docs/cjm-user.puml) | Каталог → меню → корзина из разных ресторанов → оформление с адресом → получение. Точки трения: товар недоступен, отмена рестораном, разное время приготовления |
| CJM ресторана | [`docs/cjm-restaurant.puml`](docs/cjm-restaurant.puml) | Регистрация/API-ключ → загрузка меню → polling → принятие → приготовление → готовность → управление наличием |

Рендер любой диаграммы: вставить `.puml`-файл в [plantuml.com](https://www.plantuml.com/plantuml/) или локально `plantuml docs/<file>.puml`.

## 8. Допущения MVP (упрощения)

1. **Нет авторизации пользователей** — по заданию: `client_id` передаётся в теле `POST /orders` без проверки.
2. **`orders.client_id` без FK на `users`** — наследие базовой миграции `000001`.
3. **Авторизация ресторанов — один статический API-ключ** в `X-Restaurant-API-Key`, без ротации, ролей и OAuth.
4. **Polling вместо push** — ресторан опрашивает `GET /restaurant/orders` каждые 5 секунд; вебхуков/очередей нет.
5. **Оплата, доставка, курьеры** — вне скоупа; общий `delivery_address` хранится на заказе.
6. **Нет категорий товаров, поиска, пагинации, рейтингов** — только список и CRUD меню.
7. **Отмена — только на уровне sub_order** (`cancelled`); компенсации/возвраты и частичный пересчёт на клиенте не реализованы.
8. **Цены фиксируются** в `price_at_moment` на момент заказа — последующие изменения меню историю не меняют.
9. **Сиды захардкожены** (3 ресторана, 15 товаров, 3 пользователя, ключи в открытом виде) — только для демо.

## 9. Планы масштабирования

1. **Аутентификация**: JWT/OAuth для пользователей, ротация и скоупы API-ключей ресторанов, rate limiting.
2. **Отказ от polling**: очереди (Kafka/RabbitMQ) или вебхуки/события `sub_order.status_changed`; WebSocket для live-трекинга заказа пользователем.
3. **Саги/транзакции**: создание заказа из N подзаказов — outbox + компенсирующие действия при отказе ресторана.
4. **Декомпозиция**: отдельные сервисы каталога, заказов, биллинга, уведомлений; read-модели/поиск (Elasticsearch) для меню.
5. **Доставка**: сервис курьеров, назначение курьера, гео-маршруты, SLA по времени приготовления каждого ресторана + оценка готовности всего заказа (max по sub_orders).
6. **Наблюдаемость**: tracing (OpenTelemetry), метрики, алерты на зависшие `pending`; идемпотентность `POST /orders` (Idempotency-Key).
7. **Данные**: пагинация/фильтры каталога, полнотекстовый поиск, кэширование меню (Redis), партиционирование `orders/sub_orders`.

## 10. AI-first подход

Проект разработан в AI-first парадигме (см. `PLAN.md` — детальный план-«промпт», по которому собирался проект):

- План фиксирует архитектуру до кода: ER-связи, 11 эндпоинтов OpenAPI, слои `delivery → usecase → repository`, цикл `restaurant-service`, сиды.
- ИИ использовался для генерации бойлерплейта (миграции, модели, CRUD репозиториев, хендлеры по `generated.go`), человек отвечает за архитектуру, ревью и приёмку (`go build`, `golangci-lint`, `mise run up`, сквозные curl-сценарии).
- Артефакты для проверки ИИ-результата: OpenAPI-спецификация, миграции, C4/ER-диаграммы и CJM в виде кода (PlantUML) — всё в репозитории и ревьюится как код.
