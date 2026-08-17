-- Админы для JWT-аутентификации.
CREATE TABLE IF NOT EXISTS admins (
  id            serial PRIMARY KEY,
  username      text NOT NULL UNIQUE,
  password_hash text NOT NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS trg_admins_updated_at ON admins;
CREATE TRIGGER trg_admins_updated_at BEFORE UPDATE ON admins
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ==================== МУЛЬТИЯЗЫЧНОСТЬ ====================
-- Добавляем поле language ('ru', 'en', ...) в контентные таблицы.
-- Одна и та же сущность на разных языках - это отдельная строка
-- (например, "Освещение" на ru и "Lighting" на en - два разных ряда).
-- slug теперь уникален не сам по себе, а в паре (slug, language),
-- чтобы можно было использовать одинаковый человекочитаемый slug
-- для разных языковых версий при желании.

ALTER TABLE categories ADD COLUMN IF NOT EXISTS language varchar(10) NOT NULL DEFAULT 'ru';
ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_slug_key;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'categories_slug_language_key') THEN
    ALTER TABLE categories ADD CONSTRAINT categories_slug_language_key UNIQUE (slug, language);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_categories_language ON categories(language);

ALTER TABLE subcategories ADD COLUMN IF NOT EXISTS language varchar(10) NOT NULL DEFAULT 'ru';
ALTER TABLE subcategories DROP CONSTRAINT IF EXISTS subcategories_slug_key;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'subcategories_slug_language_key') THEN
    ALTER TABLE subcategories ADD CONSTRAINT subcategories_slug_language_key UNIQUE (slug, language);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_subcategories_language ON subcategories(language);

ALTER TABLE products ADD COLUMN IF NOT EXISTS language varchar(10) NOT NULL DEFAULT 'ru';
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_slug_key;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'products_slug_language_key') THEN
    ALTER TABLE products ADD CONSTRAINT products_slug_language_key UNIQUE (slug, language);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_products_language ON products(language);

ALTER TABLE services ADD COLUMN IF NOT EXISTS language varchar(10) NOT NULL DEFAULT 'ru';
ALTER TABLE services DROP CONSTRAINT IF EXISTS services_slug_key;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'services_slug_language_key') THEN
    ALTER TABLE services ADD CONSTRAINT services_slug_language_key UNIQUE (slug, language);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_services_language ON services(language);
