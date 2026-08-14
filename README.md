# SolarMax API

Backend на Go + PostgreSQL для каталога: Категории → Подкатегории → Товары, плюс отдельный раздел Услуги.
Архитектура намеренно простая (без ORM, без DDD-слоёв) — чистый `net/http` + `chi` + `pgx`, чтобы было быстро развернуть и легко поддерживать.

## Стек
- Go 1.22, роутер [chi](https://github.com/go-chi/chi)
- PostgreSQL, драйвер [pgx v5](https://github.com/jackc/pgx) (без ORM, прямой SQL)
- Файлы (картинки) хранятся на диске в отдельной папке `media/`, а в БД — только ссылка на них
- Docker / docker-compose

## Структура проекта

```
cmd/api/main.go          - точка входа, поднимает БД + миграции + сервер
internal/config          - чтение переменных окружения
internal/db               - подключение к Postgres, выполнение миграций
internal/models           - структуры сущностей и DTO для запросов
internal/utils             - JSON-хелперы, транслитерация slug
internal/handlers          - весь HTTP-слой (роутинг + обработчики) по одному файлу на сущность
migrations/001_init.sql   - схема БД (выполняется автоматически при старте)
media/                      - сюда сохраняются загруженные картинки, раздаются по /media/...
```

## Как это соответствует вашим таблицам

| Ваша таблица   | Таблица в БД      | Примечание |
|----------------|-------------------|------------|
| Категория      | `categories`      | |
| под-Категория  | `subcategories`   | `category_id` - связь с категорией |
| Товар          | `products`        | `subcategory_id` - связь с подкатегорией, `content` = "Содержимое" |
| Услуги         | `services`        | отдельная сущность, без связей с категориями |
| Характеристики | `characteristics` + `characteristic_values` + `product_characteristics` | см. ниже |
| Изображения (везде) | `media` | одна общая таблица для картинок любой сущности |

### Как работают характеристики (пункт 5)

Сделано как в обычных интернет-магазинах:

1. `characteristics` — словарь характеристик, например "Цвет", "Мощность". Создаётся администратором один раз.
2. `characteristic_values` — конкретные значения для каждой характеристики, например для "Цвет": "Красный", "Синий". Тоже создаются один раз и переиспользуются.
3. `product_characteristics` — связка: у конкретного товара просто выбирается характеристика + значение из уже готовых списков (никакого свободного ввода не требуется).

Пример потока:
```
POST /api/characteristics            {"name": "Цвет"}                     -> id=1
POST /api/characteristics/1/values   {"value": "Красный"}                 -> id=10
POST /api/characteristics/1/values   {"value": "Белый"}                    -> id=11
POST /api/products/5/characteristics {"characteristic_id": 1, "value_id": 10}
```
Дальше на фронте это можно рисовать как выпадающий список "Выберите цвет".

### Изображения (пункт 4)

Общая таблица `media` с полями `entity_type` (category/subcategory/product/service) и `entity_id`.
Позволяет прикрепить сколько угодно картинок к любой из четырёх таблиц, без ALTER TABLE под каждую.
Одну картинку можно пометить как "главную" (`is_main`) — например, для обложки товара.

Загрузка:
```
POST /api/media/upload   (multipart/form-data)
  entity_type = product
  entity_id   = 5
  is_main     = true          (необязательно)
  files       = [file1.jpg, file2.jpg, ...]   можно несколько файлов сразу
```
В ответ приходит массив созданных записей с полем `url`, например `/media/product/5/98af...jpg`.
Файл потом открывается напрямую по `http://localhost:8080/media/product/5/98af...jpg`.

## Бизнес-правила (как реализованы)

1. **Нельзя удалить категорию, если есть подкатегории** — `DELETE /api/categories/{id}` считает подкатегории и возвращает `409 Conflict`, если их больше 0.
2. **Нельзя удалить подкатегорию, если есть товары** — аналогично, `DELETE /api/subcategories/{id}` проверяет товары.
3. **Товар можно скрыть без удаления** — `PATCH /api/products/{id}/status {"status":"inactive"}`. При выборке товаров на витрину фронт просто фильтрует `?status=active`. То же самое сделано и для услуг (`PATCH /api/services/{id}/status`), чтобы было единообразно.
4. **Несколько изображений в любой таблице** — общая таблица `media`, см. выше.
5. **Характеристики** — словарь + значения + связка, см. выше.

## Все эндпоинты

### Категории
- `GET    /api/categories?status=active`
- `GET    /api/categories/{id}`
- `POST   /api/categories`
- `PUT    /api/categories/{id}`
- `DELETE /api/categories/{id}`

### Подкатегории
- `GET    /api/subcategories?category_id=1&status=active`
- `GET    /api/subcategories/{id}`
- `POST   /api/subcategories`
- `PUT    /api/subcategories/{id}`
- `DELETE /api/subcategories/{id}`

### Товары
- `GET    /api/products?subcategory_id=1&status=active`
- `GET    /api/products/{id}`
- `POST   /api/products`
- `PUT    /api/products/{id}`
- `PATCH  /api/products/{id}/status`   `{"status": "inactive"}`
- `DELETE /api/products/{id}`
- `GET    /api/products/{id}/characteristics`
- `POST   /api/products/{id}/characteristics`  `{"characteristic_id":1,"value_id":10}`
- `DELETE /api/products/{id}/characteristics/{characteristicID}`

### Услуги
- `GET    /api/services?status=active`
- `GET    /api/services/{id}`
- `POST   /api/services`
- `PUT    /api/services/{id}`
- `PATCH  /api/services/{id}/status`
- `DELETE /api/services/{id}`

### Характеристики (словарь)
- `GET    /api/characteristics`
- `POST   /api/characteristics`             `{"name": "Цвет"}`
- `PUT    /api/characteristics/{id}`
- `DELETE /api/characteristics/{id}`
- `GET    /api/characteristics/{id}/values`
- `POST   /api/characteristics/{id}/values`  `{"value": "Красный"}`
- `PUT    /api/characteristics/{id}/values/{valueID}`
- `DELETE /api/characteristics/{id}/values/{valueID}`

### Медиа
- `POST   /api/media/upload`   (multipart, поля: entity_type, entity_id, is_main, files[])
- `GET    /api/media?entity_type=product&entity_id=5`
- `DELETE /api/media/{id}`
- `PATCH  /api/media/{id}/main`  (сделать изображение главным)

Примеры тел запроса для создания:

```json
// POST /api/categories
{ "name": "Освещение", "description": "...", "sort_order": 1, "status": "active" }
// slug сгенерируется автоматически из name (можно передать свой)

// POST /api/subcategories
{ "category_id": 1, "name": "Внутреннее освещение" }

// POST /api/products
{ "subcategory_id": 1, "name": "Светодиодная лампа", "price": 120000, "content": "..." }

// POST /api/services
{ "name": "Монтаж солнечных панелей", "advantages": "Гарантия 10 лет" }
```

## Запуск через Docker (рекомендуется)

```bash
docker compose up --build
```
Поднимутся:
- `db`  — Postgres на `localhost:5432` (логин/пароль/база: `solarmax`/`solarmax`/`solarmax`)
- `api` — сервер на `http://localhost:8080`

Миграции выполняются автоматически при старте приложения (файл `migrations/001_init.sql`, безопасно перезапускать).
Загруженные файлы сохраняются в docker volume `media_data` (не теряются при пересборке).

## Деплой на Render (или похожий хостинг)

Render запускает приложение через ваш `Dockerfile` — отдельная база нужна как **Render Postgres** (или внешняя).

1. Создайте на Render **PostgreSQL** (Render → New → PostgreSQL). Render выдаст готовую строку подключения `Internal Database URL` (или `External Database URL`).
2. Создайте **Web Service** из этого репозитория (Render сам найдёт `Dockerfile`).
3. В Environment Variables веб-сервиса добавьте:

   | Переменная | Значение |
   |---|---|
   | `DATABASE_URL` | вставьте `Internal Database URL` из шага 1 |
   | `DB_SSLMODE` | `require` *(нужно, если не используете DATABASE_URL, а собираете строку из DB_HOST/USER/... вручную)* |
   | `MEDIA_DIR` | `/app/media` (можно не задавать — это значение по умолчанию в Dockerfile) |

   `PORT` задавать не нужно — Render сам прокидывает свою переменную `PORT`, и сервер уже её слушает.

4. **Важно про картинки:** без диска Render использует эфемерную файловую систему — всё, что лежит в `/app/media`, стирается при каждом редеплое/рестарте контейнера. Чтобы файлы не терялись:
   - либо подключите **Render Disk** (платная persistent-функция) и примонтируйте его на `/app/media`;
   - либо (надёжнее для продакшена) вынесите хранение картинок во внешнее объектное хранилище (S3 / Cloudflare R2 / DO Spaces) — сейчас в проекте это не реализовано, дайте знать, если нужно — добавлю.

## Запуск локально без Docker

1. Поднимите свой Postgres, создайте базу `solarmax`.
2. Скопируйте `.env.example` → `.env` и поправьте под себя.
3. Установите зависимости и запустите:
```bash
go mod tidy
go run ./cmd/api
```

## Проверка

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/categories
```
