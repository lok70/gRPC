# gRPC Auth Service

Cервис аутентификации на Go.

## Возможности
- `Register`: регистрация пользователя
- `Login`: проверка логина/пароля и выдача JWT
- `IsAdmin`: проверка административной роли
- SQLite-хранилище + SQL-миграции
- gRPC Reflection (удобная ручная проверка через `grpcurl`)

## Стек
- Go 1.21+
- gRPC
- JWT (`github.com/golang-jwt/jwt/v5`)
- SQLite (pure Go driver `modernc.org/sqlite`)
- Migrations (`golang-migrate`)
- Testify + интеграционные тесты

## Структура проекта
- `cmd/sso` - запуск gRPC сервиса
- `cmd/migrator` - применение миграций
- `internal/services/auth` - бизнес-логика авторизации
- `internal/grpc/auth` - gRPC handlers
- `internal/storage/sqlite` - слой работы с БД
- `migrations` - SQL-миграции
- `tests` - интеграционные тесты


По умолчанию сервис слушает порт `44044`.

## Запуск через Docker
- Сборка и старт:
  - `docker compose up -d --build`
- Логи:
  - `docker compose logs -f --tail 50`
- Остановка:
  - `docker compose down`

Что делает контейнер:
- применяет миграции;
- запускает gRPC-сервис;
- хранит БД в отдельном docker volume `grpc_auth_data`.

## Проверка API через grpcurl
После запуска сервиса:

1. Посмотреть доступные сервисы:
   - `grpcurl -plaintext localhost:44044 list`
2. Посмотреть методы Auth-сервиса:
   - `grpcurl -plaintext localhost:44044 list auth.Auth`
3. Зарегистрировать пользователя:
   - `grpcurl -plaintext -d '{"email":"demo@example.com","password":"StrongPass123"}' localhost:44044 auth.Auth/Register`
4. Войти и получить JWT:
   - `grpcurl -plaintext -d '{"email":"demo@example.com","password":"StrongPass123","appId":1}' localhost:44044 auth.Auth/Login`
5. Проверить роль пользователя:
   - `grpcurl -plaintext -d '{"userId":1}' localhost:44044 auth.Auth/IsAdmin`

Если `grpcurl` не установлен локально, можно проверить из Docker:
- `docker run --rm fullstorydev/grpcurl -plaintext host.docker.internal:44044 list`

## Конфигурация
Путь к конфигу задается:
- флагом `--config`
- или переменной окружения `CONFIG_PATH`

Конфиги:
- `config/config.yaml` - локальная разработка
- `config/prod.yaml` - прод-сценарий
- `config/docker.yaml` - запуск в контейнере

## Тесты
Запуск:
- `go test ./... -v`

Интеграционные тесты самодостаточны:
- поднимают gRPC-сервер сами;
- создают отдельную временную SQLite БД;
- применяют миграции перед прогоном.
