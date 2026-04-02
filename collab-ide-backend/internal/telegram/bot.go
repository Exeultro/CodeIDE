package telegram

import (
	"context"
	"log"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Bot struct {
	bot      *tgbot.Bot
	token    string
	chatID   string
	handlers *Handlers
}

func NewBot(token, chatID string, handlers *Handlers) *Bot {
	return &Bot{
		token:    token,
		chatID:   chatID,
		handlers: handlers,
	}
}

func (b *Bot) Start(ctx context.Context) error {
	opts := []tgbot.Option{
		tgbot.WithDefaultHandler(b.defaultHandler),
	}

	bot, err := tgbot.New(b.token, opts...)
	if err != nil {
		return err
	}
	b.bot = bot

	// Регистрируем команды
	b.registerCommands()

	log.Println("🤖 Telegram bot started")

	// Запускаем бота
	go b.bot.Start(ctx)
	return nil
}

func (b *Bot) registerCommands() {
	if b.bot == nil {
		return
	}
	// Команды бота
	b.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/start", tgbot.MatchTypeExact, b.handlers.Start)
	b.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/help", tgbot.MatchTypeExact, b.handlers.Help)
	b.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/login", tgbot.MatchTypePrefix, b.handlers.Login) // 👈 ДОБАВИТЬ
	b.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/sessions", tgbot.MatchTypeExact, b.handlers.MySessions)
	b.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/leaderboard", tgbot.MatchTypePrefix, b.handlers.Leaderboard)
	b.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/events", tgbot.MatchTypePrefix, b.handlers.Events)
	b.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/hint", tgbot.MatchTypePrefix, b.handlers.Hint)
	b.bot.RegisterHandler(tgbot.HandlerTypeMessageText, "/code", tgbot.MatchTypePrefix, b.handlers.GetCode)

}

func (b *Bot) defaultHandler(ctx context.Context, bot *tgbot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "❌ Неизвестная команда. Используйте /help для списка команд.",
	})
}

func (b *Bot) SendNotification(message string) {
	if b.chatID == "" || b.bot == nil {
		return
	}

	ctx := context.Background()
	_, err := b.bot.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: b.chatID,
		Text:   message,
	})
	if err != nil {
		log.Printf("Failed to send telegram notification: %v", err)
	}
}

func (b *Bot) Stop() {
	if b.bot != nil {
		b.bot.Close(context.Background())
	}
}
