# SolarMax API

Backend на Go + PostgreSQL для каталога: Категории → Подкатегории → Товары, плюс отдельный раздел Услуги.
Архитектура намеренно простая (без ORM, без DDD-слоёв) — чистый `net/http` + `chi` + `pgx`.

## Стек
- Go 1.22, роутер [chi](https://github.com/go-chi/chi)
- PostgreSQL, драйвер [pgx v5](https://github.com/jackc/pgx) (без ORM, прямой SQL)
- JWT-авторизация ([golang-jwt/jwt](https://github.com/golang-jwt/jwt)) + bcrypt для паролей
- Мультиязычность через jsonb-колонку `translations`
- Файлы (картинки) хранятся на диске в отдельной папке `media/`, а в БД — только ссылка на них
- Docker / docker-compose

## Структура проекта

```
cmd/api/main.go          - точка входа: БД, миграции, сид админа, сервер
internal/config          - чтение переменных окружения
internal/db               - подключение к Postgres, выполнение миграций
internal/authutil          - JWT-токены, хэширование паролей, сид админа
internal/models           - структуры сущностей и DTO для запросов
internal/utils             - JSON-хелперы, jsonb-хелперы, транслитерация slug
internal/handlers          - весь HTTP-слой (роутинг + обработчики)
migrations/                 - схема БД (выполняется автоматически при старте, по порядку)
media/                      - сюда сохраняются загруженные картинки, раздаются по /media/...
```

## Авторизация (JWT)

**Все GET-запросы открыты всем** (для витрины на фронте). **Все изменяющие запросы (POST/PUT/PATCH/DELETE) требуют токен** — нужно для админ-панели.

При первом старте приложения автоматически создаётся администратор, если в таблице `admins` ещё никого нет:
```
логин:  admin
пароль: admin
```
**Обязательно смените пароль после первого входа.**

### Получить токен
```
POST /api/auth/login
{ "username": "admin", "password": "admin" }
```
Ответ:
```json
{
  "token": "eyJhbGciOi...",
  "expires_at": "2026-08-26T12:00:00Z",
  "username": "admin"
}
```

### Использовать токен
На все изменяющие запросы добавляйте заголовок:
```
Authorization: Bearer eyJhbGciOi...
```

### Остальные auth-эндпоинты
- `GET  /api/auth/me` (защищён) — проверить валидность токена.
- `POST /api/auth/change-password` (защищён):
  ```json
  { "old_password": "admin", "new_password": "новый-надёжный-пароль" }
  ```

### Срок жизни токена
По умолчанию — **24 часа** (`JWT_EXPIRY_HOURS`). Практичный баланс для простой админки без refresh-токенов.

**Обязательно смените `JWT_SECRET`** на свой длинный случайный секрет в переменных окружения продакшена — дефолтное значение годится только для локальной разработки.

## Мультиязычность

Каждая сущность (категория, подкатегория, товар, услуга) — это **одна строка** с полем `language`, внутри которого лежат переводы на разные языки. Никаких дублирующих строк на каждый язык — один объект, любое количество переводов, можно добавлять новые языки в любой момент.

Формат (пример для товара):
```json
{
  "id": 5,
  "name": "Fuel Card",
  "price": 120000,
  "slug": "fuel-card",
  "status": "active",
  "language": {
    "ru": { "name": "Топливная карта", "description": "Описание на русском" },
    "en": { "name": "Fuel Card", "description": "Description in English" },
    "uz": { "name": "Yoqilg'i kartasi", "description": "O'zbekcha tavsif" }
  }
}
```

- **`name` на верхнем уровне** — служебное/референсное имя (используется для генерации slug и как fallback), не завязано на конкретный язык.
- **`language`** — объект `{код_языка: {...переводимые поля...}}`. Можно передать один язык, все сразу, или добавить новый позже простым `PUT`-обновлением с расширенным объектом `language` (старые языки не потеряются, если прислать их тоже — API просто перезаписывает весь объект `language` целиком при каждом PUT/POST, так что при обновлении присылайте полный набор нужных переводов).
- Переводимые поля у каждой сущности:
  - Категория / Подкатегория: `name`, `description`
  - Товар: `name`, `description`, `content`
  - Услуга: `name`, `description`, `content`, `advantages`
- Нетранслируемые поля (slug, price, status, sort_order, связи `category_id`/`subcategory_id`, картинки) остаются общими для всех языков — они одни на сущность.
- Коды языков (`ru`, `en`, `uz`, ...) никак не валидируются на бэкенде — фронт сам решает, какие стандарты использовать.
- Фильтр `?language=ru` в GET-списках вернёт только те записи, у которых **есть** перевод на этот язык.

### Пример: добавить украинский перевод к уже существующему товару
```
PUT /api/products/5
{
  "subcategory_id": 1,
  "name": "Fuel Card",
  "price": 120000,
  "language": {
    "ru": { "name": "Топливная карта", "description": "Описание на русском" },
    "en": { "name": "Fuel Card", "description": "Description in English" },
    "uz": { "name": "Yoqilg'i kartasi", "description": "O'zbekcha tavsif" },
    "uk": { "name": "Паливна картка", "description": "Опис українською" }
  }
}
```
(так как PUT перезаписывает `language` целиком — присылайте все языки, которые должны остаться)

## Как это соответствует вашим таблицам

| Ваша таблица   | Таблица в БД      | Примечание |
|----------------|-------------------|------------|
| Категория      | `categories`      | + `translations` (jsonb) |
| под-Категория  | `subcategories`   | `category_id` - связь с категорией, + `translations` |
| Товар          | `products`        | `subcategory_id` - связь с подкатегорией, + `translations` |
| Услуги         | `services`        | отдельная сущность, + `translations` |
| Характеристики | `characteristics` + `characteristic_values` + `product_characteristics` | см. ниже |
| Изображения (везде) | `media` | одна общая таблица для картинок любой сущности |
| Админы         | `admins`          | для JWT-авторизации |

### Как работают характеристики

1. `characteristics` — словарь, например "Цвет", "Мощность". Создаётся один раз.
2. `characteristic_values` — значения для каждой характеристики, например "Красный", "Синий".
3. `product_characteristics` — связка: у товара выбирается характеристика + значение из готовых списков.

(характеристики пока без мультиязычности — если понадобится перевод названий характеристик/значений, дайте знать, добавлю по тому же принципу)

### Изображения

Общая таблица `media` (`entity_type` + `entity_id`) — сколько угодно картинок на любую из четырёх сущностей, одна может быть помечена как главная (`is_main`).

## Бизнес-правила

1. **Нельзя удалить категорию, если есть подкатегории** — `409 Conflict`.
2. **Нельзя удалить подкатегорию, если есть товары** — `409 Conflict`.
3. **Товар/услугу можно скрыть без удаления** — `PATCH /api/products/{id}/status` и `PATCH /api/services/{id}/status`, `{"status":"inactive"}`.
4. **Несколько изображений в любой таблице** — общая таблица `media`.
5. **Характеристики** — словарь + значения + связка.

## Все эндпоинты

`🔒` = требует `Authorization: Bearer <token>`

### Авторизация
- `POST   /api/auth/login`
- `GET    /api/auth/me` 🔒
- `POST   /api/auth/change-password` 🔒

### Категории
- `GET    /api/categories?status=active&language=ru`
- `GET    /api/categories/{id}`
- `POST   /api/categories` 🔒
- `PUT    /api/categories/{id}` 🔒
- `DELETE /api/categories/{id}` 🔒

### Подкатегории
- `GET    /api/subcategories?category_id=1&status=active&language=ru`
- `GET    /api/subcategories/{id}`
- `POST   /api/subcategories` 🔒
- `PUT    /api/subcategories/{id}` 🔒
- `DELETE /api/subcategories/{id}` 🔒

### Товары
- `GET    /api/products?subcategory_id=1&status=active&language=ru`
- `GET    /api/products/{id}`
- `POST   /api/products` 🔒
- `PUT    /api/products/{id}` 🔒
- `PATCH  /api/products/{id}/status` 🔒  `{"status": "inactive"}`
- `DELETE /api/products/{id}` 🔒
- `GET    /api/products/{id}/characteristics`
- `POST   /api/products/{id}/characteristics` 🔒  `{"characteristic_id":1,"value_id":10}`
- `DELETE /api/products/{id}/characteristics/{characteristicID}` 🔒

### Услуги
- `GET    /api/services?status=active&language=ru`
- `GET    /api/services/{id}`
- `POST   /api/services` 🔒
- `PUT    /api/services/{id}` 🔒
- `PATCH  /api/services/{id}/status` 🔒
- `DELETE /api/services/{id}` 🔒

### Характеристики (словарь)
- `GET    /api/characteristics`
- `POST   /api/characteristics` 🔒  `{"name": "Цвет"}`
- `PUT    /api/characteristics/{id}` 🔒
- `DELETE /api/characteristics/{id}` 🔒
- `GET    /api/characteristics/{id}/values`
- `POST   /api/characteristics/{id}/values` 🔒  `{"value": "Красный"}`
- `PUT    /api/characteristics/{id}/values/{valueID}` 🔒
- `DELETE /api/characteristics/{id}/values/{valueID}` 🔒

### Медиа
- `POST   /api/media/upload` 🔒  (multipart: entity_type, entity_id, is_main, files[])
- `GET    /api/media?entity_type=product&entity_id=5`
- `DELETE /api/media/{id}` 🔒
- `PATCH  /api/media/{id}/main` 🔒

## Запуск через Docker (рекомендуется)

```bash
docker compose up --build
```
Поднимутся:
- `db`  — Postgres на `localhost:5432` (логин/пароль/база: `solarmax`/`solarmax`/`solarmax`)
- `api` — сервер на `http://localhost:8080`

Миграции и сид админа выполняются автоматически при старте (можно безопасно перезапускать).

## Переменные окружения

| Переменная | Обязательна | Описание |
|---|---|---|
| `DATABASE_URL` | нет* | готовая строка подключения (например, от Render Postgres) |
| `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME`/`DB_SSLMODE` | нет* | альтернатива DATABASE_URL, собираются в строку вручную |
| `MEDIA_DIR` | нет | путь для файлов, по умолчанию `./media` |
| `PORT` | нет | порт сервера, по умолчанию `8080` (на Render подставляется автоматически) |
| `JWT_SECRET` | **да, на проде** | секрет для подписи токенов, обязательно смените |
| `JWT_EXPIRY_HOURS` | нет | срок жизни токена в часах, по умолчанию `24` |

\* нужна хотя бы одна из двух схем подключения к БД.

## Запуск локально без Docker

1. Поднимите свой Postgres, создайте базу `solarmax`.
2. Скопируйте `.env.example` → `.env` и поправьте под себя.
3. Установите зависимости и запустите:
```bash
go mod tidy
go run ./cmd/api
```

## Деплой на Render

1. Создайте **PostgreSQL** на Render, скопируйте `Internal Database URL`.
2. Создайте **Web Service** из репозитория (Render сам найдёт `Dockerfile`).
3. В Environment добавьте:
   - `DATABASE_URL` = Internal Database URL из шага 1
   - `JWT_SECRET` = свой длинный случайный секрет
4. `PORT` не задавайте — Render подставит сам.
5. Диск на Render эфемерный — без подключения persistent disk загруженные картинки не переживут редеплой.

## Проверка

```bash
curl https://<ваш-адрес>/health
curl https://<ваш-адрес>/api/categories
curl -X POST https://<ваш-адрес>/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin"}'
```
