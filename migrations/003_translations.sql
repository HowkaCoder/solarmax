-- ==================== МУЛЬТИЯЗЫЧНОСТЬ v2 ====================
-- Вместо отдельной строки на каждый язык - одна сущность = одна строка,
-- а все переводимые тексты (name/description/content/...) хранятся
-- в jsonb-колонке translations, например:
--   { "ru": {"name": "Топливная карта", "description": "..."},
--     "en": {"name": "Fuel Card", "description": "..."} }
-- Нетранслируемые поля (slug, price, status, sort_order, связи) остаются
-- обычными колонками таблицы, как и раньше.

-- ---------- КАТЕГОРИИ ----------
DO $$
BEGIN
  -- Если это апгрейд с более ранней версии (была varchar-колонка language) -
  -- один раз переносим существующие данные в translations.
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='categories' AND column_name='language') THEN
    ALTER TABLE categories ADD COLUMN IF NOT EXISTS translations jsonb NOT NULL DEFAULT '{}'::jsonb;
    UPDATE categories
      SET translations = jsonb_build_object(
        COALESCE(NULLIF(language, ''), 'ru'),
        jsonb_build_object('name', name, 'description', COALESCE(description, ''))
      )
      WHERE translations = '{}'::jsonb;
    ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_slug_language_key;
    ALTER TABLE categories DROP COLUMN IF EXISTS language;
  END IF;
END $$;

ALTER TABLE categories ADD COLUMN IF NOT EXISTS translations jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE categories DROP COLUMN IF EXISTS description;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'categories_slug_key') THEN
    ALTER TABLE categories ADD CONSTRAINT categories_slug_key UNIQUE (slug);
  END IF;
END $$;

-- ---------- ПОДКАТЕГОРИИ ----------
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='subcategories' AND column_name='language') THEN
    ALTER TABLE subcategories ADD COLUMN IF NOT EXISTS translations jsonb NOT NULL DEFAULT '{}'::jsonb;
    UPDATE subcategories
      SET translations = jsonb_build_object(
        COALESCE(NULLIF(language, ''), 'ru'),
        jsonb_build_object('name', name, 'description', COALESCE(description, ''))
      )
      WHERE translations = '{}'::jsonb;
    ALTER TABLE subcategories DROP CONSTRAINT IF EXISTS subcategories_slug_language_key;
    ALTER TABLE subcategories DROP COLUMN IF EXISTS language;
  END IF;
END $$;

ALTER TABLE subcategories ADD COLUMN IF NOT EXISTS translations jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE subcategories DROP COLUMN IF EXISTS description;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'subcategories_slug_key') THEN
    ALTER TABLE subcategories ADD CONSTRAINT subcategories_slug_key UNIQUE (slug);
  END IF;
END $$;

-- ---------- ТОВАРЫ ----------
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='products' AND column_name='language') THEN
    ALTER TABLE products ADD COLUMN IF NOT EXISTS translations jsonb NOT NULL DEFAULT '{}'::jsonb;
    UPDATE products
      SET translations = jsonb_build_object(
        COALESCE(NULLIF(language, ''), 'ru'),
        jsonb_build_object('name', name, 'description', COALESCE(description, ''), 'content', COALESCE(content, ''))
      )
      WHERE translations = '{}'::jsonb;
    ALTER TABLE products DROP CONSTRAINT IF EXISTS products_slug_language_key;
    ALTER TABLE products DROP COLUMN IF EXISTS language;
  END IF;
END $$;

ALTER TABLE products ADD COLUMN IF NOT EXISTS translations jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE products DROP COLUMN IF EXISTS description;
ALTER TABLE products DROP COLUMN IF EXISTS content;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'products_slug_key') THEN
    ALTER TABLE products ADD CONSTRAINT products_slug_key UNIQUE (slug);
  END IF;
END $$;

-- ---------- УСЛУГИ ----------
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='services' AND column_name='language') THEN
    ALTER TABLE services ADD COLUMN IF NOT EXISTS translations jsonb NOT NULL DEFAULT '{}'::jsonb;
    UPDATE services
      SET translations = jsonb_build_object(
        COALESCE(NULLIF(language, ''), 'ru'),
        jsonb_build_object('name', name, 'description', COALESCE(description, ''), 'content', COALESCE(content, ''), 'advantages', COALESCE(advantages, ''))
      )
      WHERE translations = '{}'::jsonb;
    ALTER TABLE services DROP CONSTRAINT IF EXISTS services_slug_language_key;
    ALTER TABLE services DROP COLUMN IF EXISTS language;
  END IF;
END $$;

ALTER TABLE services ADD COLUMN IF NOT EXISTS translations jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE services DROP COLUMN IF EXISTS description;
ALTER TABLE services DROP COLUMN IF EXISTS content;
ALTER TABLE services DROP COLUMN IF EXISTS advantages;

DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'services_slug_key') THEN
    ALTER TABLE services ADD CONSTRAINT services_slug_key UNIQUE (slug);
  END IF;
END $$;
