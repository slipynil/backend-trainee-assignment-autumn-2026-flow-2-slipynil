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
