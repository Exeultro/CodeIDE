package telegram

import (
	"collab-ide-backend/internal/repository"
	"fmt"
)

type Notifier struct {
	bot *Bot
	db  *repository.PostgresRepo
}

func NewNotifier(bot *Bot, db *repository.PostgresRepo) *Notifier {
	return &Notifier{
		bot: bot,
		db:  db,
	}
}

// NotifyJoin уведомляет о входе участника
func (n *Notifier) NotifyJoin(sessionID, sessionName, username string) {
	message := fmt.Sprintf(
		"👋 *%s* присоединился к сессии *%s*",
		username, sessionName,
	)
	n.bot.SendNotification(message)
}

// NotifyLeave уведомляет о выходе участника
func (n *Notifier) NotifyLeave(sessionID, sessionName, username string) {
	message := fmt.Sprintf(
		"👋 *%s* покинул сессию *%s*",
		username, sessionName,
	)
	n.bot.SendNotification(message)
}

// NotifySave уведомляет о сохранении кода
func (n *Notifier) NotifySave(sessionID, sessionName, username string, version int64) {
	message := fmt.Sprintf(
		"💾 *%s* сохранил код в сессии *%s* (версия %d)",
		username, sessionName, version,
	)
	n.bot.SendNotification(message)
}

// NotifyAIReview уведомляет о новом AI ревью
func (n *Notifier) NotifyAIReview(sessionID, sessionName string, message string) {
	msg := fmt.Sprintf(
		"🤖 *AI ревью* для сессии *%s*:\n\n%s",
		sessionName, message[:min(200, len(message))],
	)
	n.bot.SendNotification(msg)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
