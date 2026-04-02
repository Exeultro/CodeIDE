package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"collab-ide-backend/internal/repository"
)

type Handlers struct {
	db        *repository.PostgresRepo
	redis     *redis.Client
	snapRepo  *repository.SnapshotRepo
	scoreRepo *repository.ScoreRepo
	aiClient  interface {
		GetHint(ctx context.Context, code string) (string, error)
	}
}

func NewHandlers(
	db *repository.PostgresRepo,
	redis *redis.Client,
	snapRepo *repository.SnapshotRepo,
	scoreRepo *repository.ScoreRepo,
	aiClient interface {
		GetHint(ctx context.Context, code string) (string, error)
	},
) *Handlers {
	return &Handlers{
		db:        db,
		redis:     redis,
		snapRepo:  snapRepo,
		scoreRepo: scoreRepo,
		aiClient:  aiClient,
	}
}

// Start - команда /start
func (h *Handlers) Start(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	text := `🤖 *CollabIDE Telegram Bot*

Привет! Я помогу тебе следить за твоими сессиями.

*Доступные команды:*
/help - показать это сообщение
/sessions - список моих сессий
/leaderboard [session_id] - рейтинг участников сессии
/events [session_id] - последние события в сессии
/hint [session_id] - получить подсказку AI
/code [session_id] - получить текущий код из сессии

Для работы с сессией нужно указать её ID после команды.`

	bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: "Markdown",
	})
}

// Help - команда /help
func (h *Handlers) Help(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	h.Start(ctx, bot, update)
}

// MySessions - команда /sessions
func (h *Handlers) MySessions(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	// Получаем токен из Redis
	tokenKey := fmt.Sprintf("tg_token:%d", update.Message.Chat.ID)
	token, err := h.redis.Get(ctx, tokenKey).Result()
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "❌ Вы не авторизованы. Используйте `/login username password`",
			ParseMode: "Markdown",
		})
		return
	}

	// Запрос к API
	req, _ := http.NewRequest("GET", "http://backend:8080/api/user/sessions", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка получения сессий",
		})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	data := result["data"].([]interface{})
	if len(data) == 0 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "📭 У вас пока нет сессий",
		})
		return
	}

	var sessions []string
	for _, s := range data {
		session := s.(map[string]interface{})
		sessions = append(sessions, fmt.Sprintf(
			"📁 *%s*\n   ID: `%s`\n   Язык: %v\n   Создана: %v",
			session["name"], session["id"], session["language"], session["created_at"],
		))
	}

	text := "📋 *Мои сессии:*\n\n" + strings.Join(sessions, "\n\n")
	bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: "Markdown",
	})
}

// Leaderboard - команда /leaderboard [session_id]
func (h *Handlers) Leaderboard(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "⚠️ Укажите ID сессии: `/leaderboard session_id`",
			ParseMode: "Markdown",
		})
		return
	}

	sessionIDStr := args[1]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Неверный формат ID сессии",
		})
		return
	}

	items, err := h.scoreRepo.Leaderboard(ctx, sessionID)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Сессия не найдена или ошибка",
		})
		return
	}

	if len(items) == 0 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🏆 В этой сессии пока нет участников",
		})
		return
	}

	var lines []string
	for i, item := range items {
		medal := ""
		switch i {
		case 0:
			medal = "🥇 "
		case 1:
			medal = "🥈 "
		case 2:
			medal = "🥉 "
		default:
			medal = fmt.Sprintf("%d. ", i+1)
		}
		lines = append(lines, fmt.Sprintf("%s%s — %d очков", medal, item["display_name"], item["points"]))
	}

	text := "🏆 *Рейтинг участников:*\n\n" + strings.Join(lines, "\n")
	bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: "Markdown",
	})
}

// Events - команда /events [session_id]
func (h *Handlers) Events(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "⚠️ Укажите ID сессии: `/events session_id`",
			ParseMode: "Markdown",
		})
		return
	}

	sessionIDStr := args[1]
	_, err := uuid.Parse(sessionIDStr)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Неверный формат ID сессии",
		})
		return
	}

	rows, err := h.db.Pool.Query(ctx,
		`SELECT event_type, details, created_at FROM session_events 
		 WHERE session_id = $1 ORDER BY created_at DESC LIMIT 10`,
		sessionIDStr)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка получения событий",
		})
		return
	}
	defer rows.Close()

	var events []string
	for rows.Next() {
		var eventType string
		var detailsJSON []byte
		var createdAt time.Time
		rows.Scan(&eventType, &detailsJSON, &createdAt)

		var details map[string]interface{}
		json.Unmarshal(detailsJSON, &details)

		username := "system"
		if u, ok := details["username"].(string); ok {
			username = u
		}

		icon := "📌"
		switch eventType {
		case "join":
			icon = "👋"
		case "leave":
			icon = "👋"
		case "save":
			icon = "💾"
		}

		events = append(events, fmt.Sprintf("%s %s: %s", icon, username, eventType))
	}

	if len(events) == 0 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "📭 В этой сессии пока нет событий",
		})
		return
	}

	text := "📋 *Последние события:*\n\n" + strings.Join(events, "\n")
	bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: "Markdown",
	})
}

// Hint - команда /hint [session_id]
func (h *Handlers) Hint(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "⚠️ Укажите ID сессии: `/hint session_id`",
			ParseMode: "Markdown",
		})
		return
	}

	sessionIDStr := args[1]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Неверный формат ID сессии",
		})
		return
	}

	content, _, err := h.snapRepo.LoadLatest(sessionID)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Сессия не найдена",
		})
		return
	}

	if h.aiClient == nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🤖 AI-навигатор временно недоступен",
		})
		return
	}

	hint, err := h.aiClient.GetHint(ctx, content)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка получения подсказки",
		})
		return
	}

	bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      fmt.Sprintf("💡 *AI-подсказка:*\n\n%s", hint),
		ParseMode: "Markdown",
	})
}

// GetCode - команда /code [session_id]
func (h *Handlers) GetCode(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	args := strings.Fields(update.Message.Text)
	if len(args) < 2 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "⚠️ Укажите ID сессии: `/code session_id`",
			ParseMode: "Markdown",
		})
		return
	}

	sessionIDStr := args[1]
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Неверный формат ID сессии",
		})
		return
	}

	content, version, err := h.snapRepo.LoadLatest(sessionID)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Сессия не найдена",
		})
		return
	}

	preview := content
	if len(preview) > 500 {
		preview = preview[:500] + "\n\n... (код обрезан)"
	}

	text := fmt.Sprintf("📄 *Код сессии* (версия %d):\n\n```\n%s\n```", version, preview)
	bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: "Markdown",
	})
}

// getUserByTelegramID получает пользователя по telegram_id
func (h *Handlers) getUserByTelegramID(telegramID int64) (string, error) {
	var userID string
	err := h.db.Pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE telegram_id = $1`, telegramID).Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

// Login - команда /login username password
func (h *Handlers) Login(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	args := strings.Fields(update.Message.Text)
	if len(args) < 3 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "⚠️ Используйте: `/login username password`",
			ParseMode: "Markdown",
		})
		return
	}

	username := args[1]
	password := args[2]

	// Авторизация через API
	loginBody := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	resp, err := http.Post("http://backend:8080/api/auth/login", "application/json", strings.NewReader(loginBody))
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Сервер недоступен",
		})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 200 {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Неверный логин или пароль",
		})
		return
	}

	data := result["data"].(map[string]interface{})
	token := data["token"].(string)
	userId := data["user_id"].(string)

	// Сохраняем токен и user_id в БД (связываем с telegram_id)
	_, err = h.db.Pool.Exec(ctx,
		`UPDATE users SET telegram_id = $1 WHERE id = $2`,
		update.Message.Chat.ID, userId)
	if err != nil {
		bot.SendMessage(ctx, &tgbot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка сохранения",
		})
		return
	}

	// Сохраняем токен в Redis (временное хранилище)
	h.redis.Set(ctx, fmt.Sprintf("tg_token:%d", update.Message.Chat.ID), token, 24*time.Hour)

	bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("✅ Добро пожаловать, %s!\nТеперь вы можете управлять своими сессиями через бота.", username),
	})
}
