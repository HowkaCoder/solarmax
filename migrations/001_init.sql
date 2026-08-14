-- SolarMax schema. Идемпотентно (можно выполнять повторно при старте приложения).

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ==================== КАТЕГОРИИ ====================
CREATE TABLE IF NOT EXISTS categories (
  id          serial PRIMARY KEY,
  name        text NOT NULL,
  description text,
  slug        text NOT NULL UNIQUE,
  sort_order  int NOT NULL DEFAULT 0,
  status      varchar(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')),
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS trg_categories_updated_at ON categories;
CREATE TRIGGER trg_categories_updated_at BEFORE UPDATE ON categories
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ==================== ПОДКАТЕГОРИИ ====================
CREATE TABLE IF NOT EXISTS subcategories (
  id          serial PRIMARY KEY,
  category_id int NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
  name        text NOT NULL,
  description text,
  slug        text NOT NULL UNIQUE,
  sort_order  int NOT NULL DEFAULT 0,
  status      varchar(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')),
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_subcategories_category_id ON subcategories(category_id);
DROP TRIGGER IF EXISTS trg_subcategories_updated_at ON subcategories;
CREATE TRIGGER trg_subcategories_updated_at BEFORE UPDATE ON subcategories
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ==================== ТОВАРЫ ====================
CREATE TABLE IF NOT EXISTS products (
  id             serial PRIMARY KEY,
  subcategory_id int NOT NULL REFERENCES subcategories(id) ON DELETE RESTRICT,
  name           text NOT NULL,
  description    text,
  content        text,              -- "Содержимое" - длинное описание/контент товара
  price          numeric(12,2),
  slug           text NOT NULL UNIQUE,
  status         varchar(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')), -- inactive = скрыт
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_products_subcategory_id ON products(subcategory_id);
DROP TRIGGER IF EXISTS trg_products_updated_at ON products;
CREATE TRIGGER trg_products_updated_at BEFORE UPDATE ON products
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ==================== УСЛУГИ ====================
CREATE TABLE IF NOT EXISTS services (
  id          serial PRIMARY KEY,
  name        text NOT NULL,
  description text,
  content     text,              -- "Содержимое"
  advantages  text,              -- "Преимущества"
  slug        text NOT NULL UNIQUE,
  status      varchar(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')),
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS trg_services_updated_at ON services;
CREATE TRIGGER trg_services_updated_at BEFORE UPDATE ON services
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ==================== ХАРАКТЕРИСТИКИ (словарь) ====================
-- Например: "Цвет", "Мощность", "Материал" — создаётся один раз админом.
CREATE TABLE IF NOT EXISTS characteristics (
  id         serial PRIMARY KEY,
  name       text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

-- Возможные значения для каждой характеристики.
-- Например для "Цвет": "Красный", "Синий", "Белый".
CREATE TABLE IF NOT EXISTS characteristic_values (
  id                 serial PRIMARY KEY,
  characteristic_id  int NOT NULL REFERENCES characteristics(id) ON DELETE CASCADE,
  value              text NOT NULL,
  created_at         timestamptz NOT NULL DEFAULT now(),
  UNIQUE (characteristic_id, value)
);
CREATE INDEX IF NOT EXISTS idx_char_values_char_id ON characteristic_values(characteristic_id);

-- Связь товара с конкретным значением характеристики.
-- Пользователь при создании товара просто выбирает характеристику и значение из списка.
CREATE TABLE IF NOT EXISTS product_characteristics (
  id                 serial PRIMARY KEY,
  product_id         int NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  characteristic_id  int NOT NULL REFERENCES characteristics(id) ON DELETE CASCADE,
  value_id           int NOT NULL REFERENCES characteristic_values(id) ON DELETE CASCADE,
  UNIQUE (product_id, characteristic_id)
);
CREATE INDEX IF NOT EXISTS idx_prod_char_product_id ON product_characteristics(product_id);

-- ==================== МЕДИА (общая таблица картинок для всех сущностей) ====================
-- entity_type: 'category' | 'subcategory' | 'product' | 'service'
-- Позволяет хранить сколько угодно изображений для любой из таблиц выше.
CREATE TABLE IF NOT EXISTS media (
  id          serial PRIMARY KEY,
  entity_type varchar(20) NOT NULL CHECK (entity_type IN ('category','subcategory','product','service')),
  entity_id   int NOT NULL,
  url         text NOT NULL,
  is_main     boolean NOT NULL DEFAULT false,
  sort_order  int NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_media_entity ON media(entity_type, entity_id);
