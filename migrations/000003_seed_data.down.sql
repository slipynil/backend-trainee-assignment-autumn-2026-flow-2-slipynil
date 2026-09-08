ALTER TABLE products DROP CONSTRAINT IF EXISTS products_restaurant_id_fkey;
TRUNCATE TABLE order_items, sub_orders, orders, products, restaurant_api_keys, restaurants, users RESTART IDENTITY CASCADE;
