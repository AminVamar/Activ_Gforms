# Activ_Gforms

API для тестирования стажёров Активбанка через Google Forms.

Стек: Go, PostgreSQL, Google Apps Script, Docker, ngrok.

## Запуск

Нужны Docker и Docker Compose.

```bash
git clone https://github.com/AminVamar/Activ_Gforms.git
cd Activ_Gforms
cp .env.example .env
```

В `.env` указать:

- `HTTP_PORT` — порт API
- `TEST_SOURCE_URL` — полный адрес, откуда GET-ом берутся тесты, например `http://10.65.10.22:8525/api/Test`
- `NGROK_AUTHTOKEN` — токен ngrok

Запуск:

```bash
docker compose up -d --build
```

Поднимутся три контейнера: `postgres`, `api`, `ngrok`. Миграции применяются автоматически.

- API: `http://localhost:<HTTP_PORT>`
- Swagger: `http://localhost:<HTTP_PORT>/swagger/index.html`
- Публичный адрес: `https://<NGROK_DOMAIN>`
- Панель ngrok: `http://localhost:4040`

Логи и остановка:

```bash
docker compose logs -f api
docker compose down
```

## Apps Script

Код лежит в `appscript/`. В свойствах скрипта заданы:

- `SHARED_SECRET` — равен `APPSCRIPT_SECRET`
- `BACKEND_WEBHOOK_URL` — `https://<NGROK_DOMAIN>/api/v1/webhook/submission`

## Локально без Docker

```bash
go run ./cmd/api
go test ./...
```

## Автор

Amin Muborakkadamov — [github.com/AminVamar](https://github.com/AminVamar)
