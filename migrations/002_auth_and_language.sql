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

-- Примечание: мультиязычность (поле language) сделана в миграции 003
-- через jsonb-колонку translations, а не через отдельную строку на язык
-- (более ранний вариант с varchar-колонкой language был заменён).
