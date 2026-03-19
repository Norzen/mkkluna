# Task Manager API

REST API сервис для управления задачами в командах с поддержкой ролевой модели, истории изменений и сложными SQL-запросами.

## Стек технологий

- **Go 1.25**
- **MySQL 8.0**
- **Redis 7**
- **Docker + Docker Compose**
- **JWT** аутентификация
- **Prometheus** метрики

## Быстрый старт

```bash
docker-compose up --build
```

Сервис будет доступен на `http://localhost:8080`.

## API эндпоинты

### Аутентификация

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/register` | Регистрация пользователя |
| POST | `/api/v1/login` | Аутентификация (JWT) |

### Команды

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/teams` | Создать команду (стать owner) |
| GET | `/api/v1/teams` | Список команд пользователя |
| POST | `/api/v1/teams/{id}/invite` | Пригласить в команду (owner/admin) |

### Задачи

| Метод | Путь | Описание |
|-------|------|----------|
| POST | `/api/v1/tasks` | Создать задачу (член команды) |
| GET | `/api/v1/tasks?team_id=1&status=todo&assignee_id=5` | Список с фильтрацией и пагинацией |
| PUT | `/api/v1/tasks/{id}` | Обновить задачу (с проверкой прав) |
| GET | `/api/v1/tasks/{id}/history` | История изменений задачи |

### Аналитика (сложные SQL-запросы)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/analytics/team-stats` | Статистика команд (JOIN 3+ таблиц + агрегация) |
| GET | `/api/v1/analytics/top-creators` | Топ-3 создателей задач по командам (оконная функция) |
| GET | `/api/v1/analytics/orphaned-tasks` | Задачи с assignee вне команды (подзапрос NOT EXISTS) |

### Служебные

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus метрики |

## Примеры запросов

### Регистрация

```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret123", "name": "Ivan"}'
```

### Логин

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret123"}'
```

### Создание команды

```bash
curl -X POST http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Backend Team", "description": "Backend developers"}'
```

### Создание задачи

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "Implement auth", "team_id": 1, "priority": "high"}'
```

### Список задач с фильтрацией

```bash
curl "http://localhost:8080/api/v1/tasks?team_id=1&status=todo&page=1&page_size=10" \
  -H "Authorization: Bearer <token>"
```

## Структура проекта

```
cmd/server/main.go              — точка входа, graceful shutdown
internal/
  config/                       — конфигурация (YAML + ENV)
  model/                        — модели данных и DTO
  repository/                   — слой работы с БД
    analytics.go                — сложные SQL-запросы
  service/                      — бизнес-логика
    email.go                    — мок email-сервис с circuit breaker
  handler/                      — HTTP-хендлеры
  middleware/                   — JWT auth, rate limiting, Prometheus
  cache/                        — Redis-кеширование (TTL 5 мин)
migrations/                     — SQL-миграции
```

## База данных

6 таблиц, 10 связей:

- `users` — пользователи
- `teams` — команды
- `team_members` — связь пользователь-команда (many-to-many + роль)
- `tasks` — задачи
- `task_history` — аудит изменений задач
- `task_comments` — комментарии к задачам

## Реализованные требования

- **Ролевая модель**: owner, admin, member
- **История изменений**: автоматический аудит при обновлении задач
- **Кеширование**: Redis с TTL 5 мин для списков задач, инвалидация при изменениях
- **Connection pooling**: настраиваемые MaxOpenConns, MaxIdleConns, ConnMaxLifetime
- **Пагинация**: LIMIT/OFFSET на уровне БД
- **Rate limiting**: 100 запросов/мин на пользователя
- **Circuit breaker**: для внешнего email-сервиса (gobreaker)
- **Graceful shutdown**: корректное завершение с таймаутом 30 сек
- **Prometheus метрики**: количество запросов, ошибок, время ответа
- **Индексы**: оптимизированные индексы для всех запросов

## Тестирование

```bash
# Unit-тесты
go test ./internal/service/... -v

# Интеграционные тесты (требуется Docker)
go test ./internal/repository/... -v

# Покрытие
go test ./internal/service/... -cover
```

- Unit-тесты на бизнес-логику: 86.2% покрытие
- Интеграционные тесты с MySQL через testcontainers

## Конфигурация

Через `config.yaml` или переменные окружения:

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `DATABASE_HOST` | Хост MySQL | localhost |
| `DATABASE_PORT` | Порт MySQL | 3306 |
| `DATABASE_USER` | Пользователь БД | taskmanager |
| `DATABASE_PASSWORD` | Пароль БД | taskmanager_pass |
| `DATABASE_NAME` | Имя БД | taskmanager |
| `REDIS_ADDR` | Адрес Redis | localhost:6379 |
| `JWT_SECRET` | Секрет для JWT | change-me-in-production-please |
