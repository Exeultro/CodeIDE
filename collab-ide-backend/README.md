## 📘 README.md

```markdown
# CollabIDE - Платформа реального времени для совместного программирования с AI-ревьюером

Веб-IDE с поддержкой совместного редактирования в реальном времени, встроенным AI-ревьюером, терминалом и Docker-песочницей для выполнения кода.

---

## 🚀 Быстрый старт

### Требования

- **Windows 10/11 Pro, Enterprise или Education** (с поддержкой WSL 2)
- **Docker Desktop** (скачать с [docker.com](https://www.docker.com/products/docker-desktop/))
- **4 GB RAM** (рекомендуется 8 GB)
- **10 GB свободного места на диске**

### Установка Docker Desktop

1. Скачайте Docker Desktop для Windows: https://www.docker.com/products/docker-desktop/
2. Установите, следуя инструкциям установщика
3. **Включите WSL 2** при установке (галочка в установщике)
4. Перезагрузите компьютер
5. Запустите Docker Desktop, дождитесь когда иконка в трее станет зелёной

---

## 🐳 Запуск проекта

### Шаг 1: Клонирование репозитория

Просто скопируйте папку с проектом в удобное место.

### Шаг 2: Запуск контейнеров

Откройте **PowerShell** в папке с проектом и выполните:

```powershell
docker-compose up -d --build
```

**Что произойдёт:**
- Скачаются образы PostgreSQL, Redis, Ollama
- Соберутся контейнеры бэкенда и Yjs сервера
- Запустятся все сервисы

Процесс может занять **3-5 минут** (зависит от скорости интернета).

### Шаг 3: Загрузка AI модели

После запуска контейнеров выполните:

```powershell
docker exec collabide-ollama ollama pull gemma3:1b
```

Это скачает AI модель **gemma3:1b** (около 500 MB). Загрузка займёт **5-10 минут**.

### Шаг 4: Проверка работы

Откройте браузер и перейдите по адресу:

```
http://localhost:8080/healthz
```

Должен прийти ответ:

```json
{"success":true,"data":{"status":"ok","timestamp":"..."}}
```

---

## 🛑 Остановка проекта

```powershell
docker-compose down
```

---

## 🔄 Перезапуск после остановки

```powershell
docker-compose up -d
```

---

## 📡 Доступные сервисы

| Сервис | URL | Описание |
|--------|-----|----------|
| Бэкенд API | http://localhost:8080 | REST API |
| WebSocket | ws://localhost:8080/ws | Real-time синхронизация |
| Yjs CRDT | ws://localhost:1234 | Совместное редактирование |
| PostgreSQL | localhost:5433 | База данных (пароль: secret) |
| Redis | localhost:6379 | Кэш |
| Ollama | http://localhost:11434 | AI модель |

---

## 🧪 Проверка работы всех сервисов

### API бэкенда

```powershell
# Регистрация пользователя
curl -X POST http://localhost:8080/api/auth/register `
  -H "Content-Type: application/json" `
  -d '{"username":"test","password":"test123"}'

# Вход
curl -X POST http://localhost:8080/api/auth/login `
  -H "Content-Type: application/json" `
  -d '{"username":"test","password":"test123"}'

# Создание сессии (нужен токен из ответа выше)
curl -X POST http://localhost:8080/api/sessions `
  -H "Content-Type: application/json" `
  -H "Authorization: Bearer <ваш_токен>" `
  -d '{"name":"Моя сессия","file_name":"main.py","language":"python"}'
```

### WebSocket

```powershell
# Установка wscat
npm install -g wscat

# Подключение к WebSocket
wscat -c "ws://localhost:8080/ws?room=test&user=123&username=test"
```

### Yjs

```powershell
wscat -c "ws://localhost:1234/yjs?room=test&user=123"
```

---

## 📁 Структура проекта

```
collab-ide-backend/
├── cmd/
│   ├── server/          # Основной сервер
│   └── yjs-server/      # Yjs WebSocket сервер
├── internal/
│   ├── api/             # REST и WebSocket API
│   ├── auth/            # JWT аутентификация
│   ├── config/          # Конфигурация
│   ├── core/            # AI и Docker-песочница
│   ├── middleware/      # CORS, JWT
│   ├── models/          # Модели данных
│   ├── repository/      # Работа с БД
│   └── telegram/        # Telegram бот (опционально)
├── docker-compose.yml
├── Dockerfile
├── Dockerfile.yjs
├── go.mod
├── go.sum
└── README.md
