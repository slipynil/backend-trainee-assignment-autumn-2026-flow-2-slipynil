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
