# SolarMax API

Версия 5.0 — бэкенд на Go **без внешней базы данных и без внешних зависимостей**.

Postgres убран: данные живут внутри процесса и сохраняются в один JSON-файл.
В `go.mod` нет ни одной библиотеки — только стандартная библиотека Go.
Итог: собирается без интернета, деплоится одним статическим бинарником,
запускается на любом бесплатном хостинге без подключённой БД.

```bash
go build ./cmd/api   # зависимостей нет, go mod download не нужен
./api                # или: go run ./cmd/api
```

По умолчанию поднимается на `http://localhost:8080`, логин/пароль — `admin` / `admin`.

---

## Как устроено хранилище

Всё лежит в памяти (обычные `map` под `sync.RWMutex`) и раз в 3 секунды
сбрасывается в `data/solarmax.json`. Запись атомарная: сначала временный файл,
потом `rename`, поэтому битого файла на диске не окажется даже при падении.
При старте файл читается обратно, при остановке (SIGTERM/Ctrl+C) делается
финальный снапшот.

Загруженные картинки складываются в `data/media/` и раздаются по `/media/...`,
как и раньше.

Если нужен режим совсем без диска — поставьте `PERSIST=false`,
тогда данные будут жить только до перезапуска.

### Бэкап и восстановление

На бесплатных тарифах диск обычно временный: при передеплое `data/` пропадает.
Для этого есть два эндпоинта (оба под токеном):

```bash
# скачать всё одним файлом
curl -H "Authorization: Bearer $TOKEN" \
     https://ваш-домен/api/admin/backup -o backup.json

# залить обратно (полностью заменяет текущие данные)
curl -X POST -H "Authorization: Bearer $TOKEN" \
     --data-binary @backup.json https://ваш-домен/api/admin/restore
```

В дамп попадают все сущности и учётка администратора (с хэшем пароля),
так что после восстановления вход работает тем же паролем.
Картинки в дамп не входят — их храните отдельно или на постоянном диске.

---

## Антизасыпание (раз в час)

Бесплатные хостинги усыпляют сервис без активности. Внутри приложения работает
фоновая горутина: первый пинг — через минуту после старта, дальше строго по
интервалу (по умолчанию час). Адрес берётся из `KEEPALIVE_URL`, а если он не
задан — автоматически из `RENDER_EXTERNAL_URL`, `PUBLIC_URL` или
`RAILWAY_PUBLIC_DOMAIN`. В крайнем случае пингуется собственный `/health`.

```
GET  /api/keepalive        # когда был последний пинг и когда будет следующий
POST /api/keepalive/ping   # запустить пинг вручную (нужен токен)
```

В логах это выглядит так:

```
keepalive: включён, интервал 1h0m0s, адреса: [https://solarmax-api.onrender.com/health]
keepalive: пинг https://solarmax-api.onrender.com/health -> 200 (312 мс)
```

**Важно и честно:** самопинг не разбудит уже уснувший сервис — спящий процесс
не может отправить запрос. На Render free обязательно добавьте внешний пингер
(cron-job.org, UptimeRobot) на `https://ваш-домен/health` раз в 10–14 минут.
Встроенный keepalive — это подстраховка для хостингов, которые смотрят на
исходящую активность, плюс удобный «пульс» в логах.

---

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `PORT` | `8080` | Порт. Render/Railway прокидывают его сами. |
| `DATA_DIR` | `./data` | Папка с данными и картинками. |
| `DATA_FILE` | `DATA_DIR/solarmax.json` | Файл снапшота. |
| `MEDIA_DIR` | `DATA_DIR/media` | Папка загруженных файлов. |
| `PERSIST` | `true` | `false` — чистый in-memory, без записи на диск. |
| `JWT_SECRET` | dev-значение | **Обязательно смените в проде.** |
| `JWT_EXPIRY_HOURS` | `24` | Время жизни токена. |
| `ADMIN_USERNAME` | `admin` | Создаётся только при первом запуске. |
| `ADMIN_PASSWORD` | `admin` | Создаётся только при первом запуске. |
| `KEEPALIVE_ENABLED` | `true` | Фоновые пинги. |
| `KEEPALIVE_INTERVAL` | `1h` | `30m`, `1h30m` или просто число минут. |
| `KEEPALIVE_URL` | автоопределение | Можно перечислить несколько через запятую. |

Админ создаётся **только если в хранилище ещё нет ни одного пользователя**.
Смена `ADMIN_PASSWORD` в окружении не меняет уже существующий пароль —
для этого есть `POST /api/auth/change-password`.

---

## API

Пути, поля JSON и коды ответов не изменились — фронтенду менять нечего.

### Авторизация
```
POST /api/auth/login             {"username","password"} -> {token, expires_at, username}
GET  /api/auth/me                (Bearer)
POST /api/auth/change-password   (Bearer) {"old_password","new_password"}
```

Все `GET` открыты всем, любые изменения требуют заголовок
`Authorization: Bearer <token>`.

### Сущности
```
/api/categories      GET, POST, GET/PUT/DELETE {id}
/api/subcategories   GET, POST, GET/PUT/DELETE {id}          ?category_id=
/api/products        GET, POST, GET/PUT/DELETE {id}          ?subcategory_id=
                     PATCH {id}/status   {"status":"inactive"}
                     GET/POST {id}/characteristics
                     DELETE {id}/characteristics/{characteristicID}
/api/services        GET, POST, GET/PUT/DELETE {id}, PATCH {id}/status
/api/object-types    GET, POST, GET/PUT/DELETE {id}, PATCH {id}/status
/api/advantages      GET, POST, GET/PUT/DELETE {id}, PATCH {id}/status
/api/characteristics GET, POST, PUT/DELETE {id}
                     GET/POST {id}/values, PUT/DELETE {id}/values/{valueID}
/api/media           POST /upload (multipart), GET ?entity_type=&entity_id=,
                     DELETE {id}, PATCH {id}/main
```

Общие фильтры списков: `?status=active|inactive`, `?language=ru|en|uz`
(отдаёт только записи, у которых есть перевод на этот язык).

### Служебное
```
GET  /health          "ok" — для хостинга и внешних пингеров
GET  /api/health      подробно: аптайм, счётчики записей, статус keepalive
GET  /api/keepalive
POST /api/keepalive/ping      (Bearer)
GET  /api/admin/backup        (Bearer)
POST /api/admin/restore       (Bearer)
```

### Правила, которые сохранены
1. Категорию нельзя удалить, пока у неё есть подкатегории — `409`.
2. Подкатегорию нельзя удалить, пока в ней есть товары — `409`.
3. Товар можно скрыть, не удаляя: `PATCH /api/products/{id}/status`.
4. Slug уникален внутри своей сущности, при конфликте — `409`.
5. Удаление товара или характеристики каскадно чистит связи и картинки.

---

## Деплой

**Docker:**
```bash
docker compose up --build
```

**Render:** в репозитории есть `render.yaml`. Блока с базой данных в нём нет.
Задайте `JWT_SECRET` и `ADMIN_PASSWORD`, `healthCheckPath` уже указан как
`/health`. На платном плане подключите постоянный диск на `/app/data` —
тогда данные и картинки переживут передеплой.

**Без Docker:** соберите `CGO_ENABLED=0 go build -o solarmax ./cmd/api` и
положите бинарник на сервер. Больше ничего ставить не нужно.

---

## Что изменилось по сравнению с версией 4.0

| Было | Стало |
|---|---|
| Postgres + `pgx` + SQL-миграции | встроенное хранилище `internal/store` |
| роутер `chi` | `http.ServeMux` из Go 1.22 |
| `golang-jwt/jwt` | свой HS256 на `crypto/hmac` (формат токена тот же) |
| `bcrypt` из `x/crypto` | PBKDF2-HMAC-SHA256 на stdlib |
| `godotenv` | `internal/dotenv` |
| `google/uuid` | имена файлов из `crypto/rand` |

Папка `migrations/` удалена — схема больше не нужна, структура данных описана
Go-типами в `internal/models`.
