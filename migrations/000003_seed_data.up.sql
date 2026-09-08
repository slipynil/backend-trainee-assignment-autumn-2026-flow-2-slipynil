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
