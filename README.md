# Go_url_shortener

HTTP API для сокращения URL, написанный на Go.

Сервис генерирует короткий код длиной 10 символов для каждого URL. При повторной отправке одного и того же URL возвращается сохранённый короткий код.

## Запуск

Нужны Docker и Docker Compose.

### 1. Клонирование репозитория

```bash
git clone https://github.com/Sneypsleep90/go-url-shortner.git
cd go-url-shortner
```

### 2. Сборка Docker-образа

```bash
docker build -t url-shortener .
```

### 3. Выбор хранилища

Приложение поддерживает два варианта хранения данных: Memory и PostgreSQL.

#### Memory

Запуск контейнера с хранением данных в памяти:

```bash
docker run --rm -p 8080:8080 \
  -e STORAGE_TYPE=memory \
  url-shortener
```

Данные теряются после остановки приложения.

#### PostgreSQL

Для постоянного хранения ссылок можно использовать PostgreSQL.

**1. Запуск базы данных:**

```bash
docker compose up -d postgres
```

**2. Проверка готовности базы данных:**

```bash
docker compose exec postgres pg_isready -U shortener -d shortener_db
```

Дождитесь сообщения:

```text
accepting connections
```

**3. Применение миграций:**

```bash
docker run --rm \
  --network go-url-shortner_default \
  -v "$PWD:/app" \
  -w /app \
  -e GOOSE_DRIVER=postgres \
  -e GOOSE_DBSTRING='postgres://shortener:shortener@postgres:5432/shortener_db?sslmode=disable' \
  golang:1.27-alpine \
  go run github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir /app/migrations up
```

**4. Запуск приложения:**

```bash
docker run --rm -p 8080:8080 \
  --network go-url-shortner_default \
  -e STORAGE_TYPE=postgres \
  -e 'DB_URL=postgres://shortener:shortener@postgres:5432/shortener_db?sslmode=disable' \
  url-shortener
```

Теперь приложение использует PostgreSQL для постоянного хранения ссылок.

## Конфигурация

Настройки приложения задаются через переменные окружения.

| Переменная | Описание | По умолчанию |
|---|---|---|
| `HTTP_ADDR` | Адрес HTTP-сервера | `:8080` |
| `STORAGE_TYPE` | Тип хранилища (`memory` или `postgres`) | `memory` |
| `DB_URL` | Строка подключения к PostgreSQL | — |

Сервер доступен по адресу `http://localhost:8080`.

## API

### Создание короткой ссылки

**Endpoint:**

```http
POST /links
```

**Запрос:**

```bash
curl -i -X POST http://localhost:8080/links \
  -H 'Content-Type: application/json' \
  -d '{"original_url":"https://example.com/page"}'
```

**Ответ — `201 Created`:**

```json
{
  "short_code": "Abc123_Xy9",
  "original_url": "https://example.com/page"
}
```

Повторный запрос с тем же URL возвращает сохранённый короткий код.

### Получение оригинального URL

**Endpoint:**

```http
GET /{short_code}
```

Вместо `Abc123_Xy9` подставьте `short_code` из ответа POST.

**Запрос:**

```bash
curl -i http://localhost:8080/Abc123_Xy9
```

**Ответ — `200 OK`:**

```json
{
  "short_code": "Abc123_Xy9",
  "original_url": "https://example.com/page"
}
```

**Ответ — `404 Not Found`:**

```json
{
  "error": "link not found"
}
```

GET возвращает JSON с оригинальным URL, а не HTTP-редирект.

### HTTP-коды ответа

| Код | Описание |
|---|---|
| `200 OK` | Оригинальный URL успешно получен |
| `201 Created` | Короткая ссылка создана |
| `400 Bad Request` | Некорректный JSON или URL |
| `404 Not Found` | Короткий код не найден |
| `500 Internal Server Error` | Внутренняя ошибка сервиса |

## Нагрузочное тестирование

Для оценки производительности API было проведено нагрузочное тестирование с использованием **Postman Performance Testing**.

Тестирование проводилось на локальной машине.

### Результаты

| Метрика | Результат |
|---|---:|
| Всего запросов | 305 460 |
| Пропускная способность | 2 541.18 RPS |
| Среднее время ответа | 2 ms |
| P90 | 3 ms |
| P95 | 4 ms |
| P99 | 17 ms |
| Error Rate | 0% |
| Failure Rate | 0% |

В ходе тестирования сервис обработал более 305 тысяч запросов со средней пропускной способностью около 2 500 запросов в секунду. Зарегистрированных ошибок и неудачных запросов не было.

### Скриншот результатов

![Результаты нагрузочного тестирования](docs/images/load-test.png)

Результаты получены на локальной машине.

## Тестирование

Для запуска тестов нужны Go 1.27+ и Make.

### Unit-тесты

```bash
make test
```

### Интеграционные тесты

Для интеграционных тестов необходима запущенная PostgreSQL и переменная окружения `TEST_DB_URL`.

```bash
TEST_DB_URL='postgres://shortener:shortener@localhost:5432/shortener_db?sslmode=disable' \
  make test-integration
```

## Makefile

| Команда | Описание |
|---|---|
| `make env` | Создаёт `.env` из `.env.example`, если файл отсутствует |
| `make test` | Запускает unit-тесты с `-race` |
| `make test-integration` | Запускает все тесты, включая интеграционные, с `-race` |

