-- ==================== ОБЪЕКТЫ (типы объектов, например "Учебные заведения") ====================
-- Блок "Мы работаем с объектами любого масштаба" - карточки фото + название.
CREATE TABLE IF NOT EXISTS object_types (
  id           serial PRIMARY KEY,
  name         text NOT NULL,
  slug         text NOT NULL UNIQUE,
  sort_order   int NOT NULL DEFAULT 0,
  status       varchar(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')),
  translations jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS trg_object_types_updated_at ON object_types;
CREATE TRIGGER trg_object_types_updated_at BEFORE UPDATE ON object_types
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ==================== ПРЕИМУЩЕСТВА ("Что мы предлагаем") ====================
-- Блок с чередующимися карточками: фото + название + текст.
CREATE TABLE IF NOT EXISTS advantages (
  id           serial PRIMARY KEY,
  name         text NOT NULL,
  slug         text NOT NULL UNIQUE,
  sort_order   int NOT NULL DEFAULT 0,
  status       varchar(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active','inactive')),
  translations jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS trg_advantages_updated_at ON advantages;
CREATE TRIGGER trg_advantages_updated_at BEFORE UPDATE ON advantages
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Разрешаем медиа-таблице хранить фото и для новых сущностей.
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_entity_type_check;
ALTER TABLE media ADD CONSTRAINT media_entity_type_check
  CHECK (entity_type IN ('category','subcategory','product','service','object_type','advantage'));
