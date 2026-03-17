# Web Server Gin

Веб-сервер с аутентификацией и управлением пользователями на Go.

## Функциональность

- **Аутентификация**: JWT с refresh токенами
- **Управление пользователями**: CRUD операции
- **gRPC API**: внутренняя коммуникация
- **Health checks**: мониторинг состояния

## API Endpoints

### Аутентификация

- `POST /api/login` - вход в систему
- `POST /api/refresh` - обновление токена
- `POST /api/logout` - выход
- `POST /api/logout-all` - выход со всех устройств

### Пользователи

- `POST /api/users` - создание пользователя
- `GET /api/users` - список пользователей
- `GET /api/users/:id` - информация о пользователе
- `PATCH /api/users/:id` - обновление пользователя
- `DELETE /api/users/:id` - удаление пользователя

### Утилиты

- `GET /api/auth/public-key` - публичный ключ JWT
- `GET /health` - проверка здоровья сервиса

## Порты

- **HTTP API**: `:8080`
- **gRPC**: `:50051`

## Запуск

### Локально

```bash
go run main.go
```

### Kubernetes

```bash
# Применение манифестов
kubectl apply -k k8s/base

# С сертификатами
kubectl apply -f k8s/cert-manager/

# Автомасштабирование
kubectl apply -f k8s/scaling/hpa.yaml
```

## Структура k8s манифестов

```
k8s/
├── base/                 # Основные манифесты
│   ├── deployment.yml
│   ├── service.yml
│   ├── ingress.yaml
│   └── secret.yaml
├── cert-manager/         # TLS сертификаты
├── scaling/              # Автомасштабирование
└── tests/               # Тестовые задания
```

## Конфигурация

- **PostgreSQL**: настраивается через secrets
- **Ingress**: с поддержкой TLS
- **HPA**: горизонтальное автомасштабирование

## Режимы

- **Debug**: логирование запросов
- **Production**: `export GIN_MODE=release`
