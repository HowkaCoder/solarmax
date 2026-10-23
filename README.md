# SolarMax API

Backend for a solar energy company website: catalog of panels, inverters and batteries, services, object types, advantages, product characteristics and media, with content in Russian, Uzbek and English and an admin panel API.

Single static binary, no external database and no third-party dependencies, only the Go standard library.

![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)
![stdlib](https://img.shields.io/badge/dependencies-0-success)
[![ci](https://github.com/HowkaCoder/solarmax/actions/workflows/ci.yml/badge.svg)](https://github.com/HowkaCoder/solarmax/actions/workflows/ci.yml)

## Why no database

The catalog is small and changes a few times a week, while the site is hosted on free plans. So data lives in memory (maps under `sync.RWMutex`) and is saved to a single JSON file every 3 seconds when something changed. Writes are atomic (temp file + `rename`), and a final snapshot is taken on shutdown.

Trade-off: everything has to fit in memory and only one instance can run. For this project it is fine; for a bigger catalog the `store` package can be replaced with a PostgreSQL implementation behind the same methods.

## Features

- Categories → subcategories → products, services, object types, advantages
- Characteristics with values, attached to products
- Translations per language, list filters `?status=active|inactive` and `?language=ru|uz|en`
- Image upload with main image and ordering
- Admin auth: HS256 JWT and PBKDF2-SHA256 passwords on the standard library
- Backup and restore of all data with one request
- Keep-alive pinger for free hostings, health endpoint with uptime and counters
- Graceful shutdown, panic recovery, request logging

Business rules: a category can't be deleted while it has subcategories, a subcategory while it has products (`409`); slugs are unique (`409`); products can be hidden without deleting.

## Getting started

```bash
cp .env.example .env
go run ./cmd/api
```

If `ADMIN_PASSWORD` is empty, a random password is generated on first start and printed to the log. If `JWT_SECRET` is empty, a random one is generated (tokens become invalid after restart), so set it in production.

Docker:

```bash
docker compose up --build
```

Render: `render.yaml` is included, health check path is `/health`.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP port |
| `DATA_DIR` | `./data` | Data and uploaded images |
| `DATA_FILE` | `DATA_DIR/solarmax.json` | Snapshot file |
| `MEDIA_DIR` | `DATA_DIR/media` | Uploaded files |
| `PERSIST` | `true` | `false` keeps data only in memory |
| `JWT_SECRET` | random | Set it in production |
| `JWT_EXPIRY_HOURS` | `24` | Token lifetime |
| `ADMIN_USERNAME`, `ADMIN_PASSWORD` | `admin`, random | Used only on first start |
| `KEEPALIVE_ENABLED`, `KEEPALIVE_INTERVAL`, `KEEPALIVE_URL` | `true`, `1h`, auto | Background pings |

## API

All `GET` requests are public, any change needs `Authorization: Bearer <token>`.

```
POST /api/auth/login             {"username","password"} -> {token, expires_at, username}
GET  /api/auth/me
POST /api/auth/change-password   {"old_password","new_password"}

/api/categories      GET, POST, GET/PUT/DELETE {id}
/api/subcategories   GET, POST, GET/PUT/DELETE {id}          ?category_id=
/api/products        GET, POST, GET/PUT/DELETE {id}          ?subcategory_id=
                     PATCH {id}/status, GET/POST {id}/characteristics
/api/services        GET, POST, GET/PUT/DELETE {id}, PATCH {id}/status
/api/object-types    GET, POST, GET/PUT/DELETE {id}, PATCH {id}/status
/api/advantages      GET, POST, GET/PUT/DELETE {id}, PATCH {id}/status
/api/characteristics GET, POST, PUT/DELETE {id}, {id}/values
/api/media           POST /upload, GET ?entity_type=&entity_id=, DELETE {id}, PATCH {id}/main

GET  /health, /api/health, /api/keepalive
GET  /api/admin/backup
POST /api/admin/restore
```

## Project structure

```
cmd/api/              entrypoint
internal/
  store/              in-memory storage with JSON snapshots
  handlers/           HTTP handlers and middleware on net/http
  authutil/           JWT and password hashing
  models/             data types
  keepalive/          background pinger
  config/, dotenv/    configuration
```

## Tests

```bash
go test -race ./...
```

Covers password hashing and tokens (including forged and expired ones), catalog rules, saving and loading snapshots, and HTTP tests for login and protected routes.
