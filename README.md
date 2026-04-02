# 🚀 CollabIDE — Совместная IDE с AI-ревьюером

Платформа реального времени для совместного программирования с встроенным AI-ревьюером, Docker-песочницей и CRDT-синхронизацией.

## 📋 О проекте

CollabIDE — это веб-IDE, где несколько разработчиков могут одновременно писать код в одном файле, видеть курсоры друг друга, общаться в реальном времени и получать AI-подсказки.

### Основные возможности

- 🤝 **Совместное редактирование** — CRDT-синхронизация через Yjs
- 🤖 **AI-ревьюер** — анализ кода и предложение исправлений (Ollama + gemma3:1b)
- 🐳 **Docker-песочница** — безопасный запуск кода в изолированных контейнерах
- 💻 **Встроенный терминал** — общий терминал для всех участников сессии
- 👥 **Участники и приглашения** — приглашайте коллег по ссылке
- 🏆 **Геймификация** — рейтинг участников за активность
- 🕵️ **Режим инкогнито** — скрывайте своё имя в рейтинге
- 📱 **Telegram бот** — управление сессиями через Telegram

---

## 🛠️ Технологический стек

| Компонент | Технология |
|-----------|------------|
| **Фронтенд** | Vue 3, TypeScript, Vite, Monaco Editor, Pinia |
| **Бэкенд** | Go 1.23, Gorilla WebSocket |
| **База данных** | PostgreSQL 15 |
| **Кэш** | Redis 7 |
| **AI** | Ollama (gemma3:1b) |
| **CRDT** | Yjs |
| **Контейнеризация** | Docker, Docker Compose |

---

## 🚀 Быстрый старт

### Требования

- **Docker Desktop** (Windows) или Docker + Docker Compose (Linux/Mac)
- **4 GB RAM** (рекомендуется 8 GB)
- **Git** (для клонирования репозитория)

### 1. Клонирование репозитория

```bash
git clone https://github.com/Exeultro/CodeIDE.git
cd CodeIDE
```
2. Запуск бэкенда
```bash
cd collab-ide-backend
docker-compose up -d --build
Первый запуск может занять 5-10 минут (скачивание образов и AI модели).
```

После запуска контейнеров необходимо скачать AI модель:

```bash
# Скачать модель gemma3:1b (около 500 MB)
docker exec collabide-ollama ollama pull gemma3:1b
```
Модель скачивается один раз. Загрузка может занять 5-10 минут в зависимости от скорости интернета.

После запуска проверьте:

```bash
curl http://localhost:8080/healthz
# Должно вернуться: {"success":true,"data":{"status":"ok",...}}
```
3. Запуск фронтенда
```bash
cd ../fronte-master
npm install
npm run dev
```
Фронтенд будет доступен по адресу: http://localhost:5173/web/

Структура проекта
```
CodeIDE/
├── collab-ide-backend/          # Бэкенд на Go
│   ├── cmd/
│   │   ├── server/              # Основной сервер
│   │   └── yjs-server/          # Yjs WebSocket сервер
│   ├── internal/
│   │   ├── api/                 # REST и WebSocket API
│   │   ├── auth/                # JWT аутентификация
│   │   ├── core/                # AI и Docker-песочница
│   │   ├── middleware/          # CORS, Rate limiting
│   │   ├── repository/          # Работа с БД
│   │   └── telegram/            # Telegram бот
│   ├── docker-compose.yml
│   ├── Dockerfile
│   └── go.mod
│
└── fronte-master/               # Фронтенд на Vue 3
    ├── src/
    │   ├── components/          # UI компоненты
    │   ├── views/               # Страницы
    │   ├── store/               # Pinia хранилища
    │   ├── services/            # API и WebSocket клиенты
    │   └── router/              # Маршрутизация
    ├── package.json
    └── vite.config.ts
```

🔧 Команды для управления
Бэкенд
```bash
cd collab-ide-backend

# Запуск всех сервисов
docker-compose up -d

# Остановка
docker-compose down

# Перезапуск с пересборкой
docker-compose up -d --build

# Просмотр логов
docker-compose logs -f

# Просмотр логов только бэкенда
docker-compose logs -f backend
```
Фронтенд
```bash
cd fronte-master

# Установка зависимостей
npm install

# Запуск в режиме разработки
npm run dev

# Сборка для продакшена
npm run build
```
🧪 Тестирование API
Регистрация пользователя
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123"}'
```
Логин
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123"}'
```
Создание сессии
```bash
curl -X POST http://localhost:8080/api/sessions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <ваш_токен>" \
  -d '{"name":"Моя сессия","file_name":"main.py","language":"python"}'
```

🤖 Telegram бот
Найдите бота в Telegram: @CollabIDEhacaton_bot
(Работает при запуске бэкенда, но надо включать ВПН!(иначе не даёт доступ к апи Телеграмма)
Авторизуйтесь: /login testuser test123

Получите список сессий: /sessions

Доступные команды:

/start — приветствие

/help — справка

/login username password — авторизация

/sessions — список моих сессий

/leaderboard <session_id> — рейтинг участников

/events <session_id> — последние события

/hint <session_id> — подсказка AI

/code <session_id> — текущий код сессии

📝 API Документация
Основные эндпоинты
Метод	Эндпоинт	Описание
POST	/api/auth/register	Регистрация
POST	/api/auth/login	Вход
POST	/api/sessions	Создать сессию
GET	/api/user/sessions	Сессии пользователя
GET	/api/sessions/{id}/content	Получить код
PUT	/api/sessions/{id}/content	Обновить код
GET	/api/sessions/{id}/ai-reviews	AI ревью
POST	/api/sessions/{id}/ai-reviews/{rid}/apply	Применить ревью
GET	/api/sessions/{id}/hint	Подсказка AI
GET	/api/sessions/{id}/leaderboard	Рейтинг
GET	/api/sessions/{id}/participants	Участники
POST	/api/sessions/{id}/invite	Пригласить

WebSocket
```text
ws://localhost:8080/ws?room={session_id}&user={user_id}&username={username}
```


Yjs WebSocket
```text
ws://localhost:1234/yjs?room={session_id}&user={user_id}
```
🐛 Устранение неполадок
Порт уже используется
```bash
# Найти процесс на порту 8080
netstat -ano | findstr :8080
# Завершить процесс
taskkill /PID <PID> /F
```
Очистка Docker
```bash
docker system prune -a -f
docker volume prune -f
```
Переустановка зависимостей фронтенда

```bash
rm -rf node_modules package-lock.json
npm install
```
Ошибка подключения к Yjs
Убедитесь, что Yjs сервер запущен:

```bash
docker ps | grep yjs
curl http://localhost:1234/health
```
