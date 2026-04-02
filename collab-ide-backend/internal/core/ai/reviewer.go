package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	URL   string
	Model string
	HTTP  *http.Client
}

func NewOllamaClient(url, model string) *OllamaClient {
	model = strings.TrimSpace(model)
	if model == "" {
		model = "gemma3:1b"
	}
	return &OllamaClient{
		URL:   strings.TrimRight(url, "/"),
		Model: model,
		HTTP:  &http.Client{Timeout: 60 * time.Second},
	}
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error"`
}

func (c *OllamaClient) generate(prompt string) (string, error) {
	reqBody := GenerateRequest{Model: c.Model, Prompt: prompt, Stream: false}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := c.HTTP.Post(c.URL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama error: %s", strings.TrimSpace(string(body)))
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Error) != "" {
		return "", fmt.Errorf("ollama error: %s", strings.TrimSpace(result.Error))
	}
	return strings.TrimSpace(result.Response), nil
}

func (c *OllamaClient) ReviewCode(code string) (string, error) {
	prompt := fmt.Sprintf(`Ты — ревьюер кода. Найди ОДНУ конкретную проблему в коде и предложи ИСПРАВЛЕННЫЙ КОД.

Код:
%s

Правила ответа:
1. Напиши проблему одной строкой
2. Напиши исправленный код (только код, без пояснений)

Формат ответа (строго!):
Проблема: [описание проблемы]
Исправление:
[исправленный код]

Пример:
Проблема: Не хватает закрывающей скобки
Исправление:
print("Hello World")

ВАЖНО: Если код правильный, напиши "Проблема: Код правильный" и не пиши исправление.`, code)

	response, err := c.generate(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}

func (c *OllamaClient) NavigatorHint(code string) (string, error) {
	prompt := fmt.Sprintf("Ты AI-навигатор для студентов. Отвечай кратко на русском. Дай 3 следующих шага по решению задачи и один маленький пример. Код:\n\n%s", code)
	return c.generate(prompt)
}

// GetHint возвращает подсказку AI для навигатора
func (c *OllamaClient) GetHint(ctx context.Context, code string) (string, error) {
	prompt := fmt.Sprintf("Ты AI-навигатор для студентов. Отвечай кратко на русском. Дай 3 следующих шага по решению задачи и один маленький пример. Код:\n\n%s", code)
	return c.generate(prompt)
}

// MergeConflicts предлагает слияние двух конфликтующих версий кода
func (c *OllamaClient) MergeConflicts(code1, code2 string) (string, error) {
	prompt := fmt.Sprintf(`
Два пользователя одновременно редактируют один и тот же участок кода.

Версия пользователя A:
%s

Версия пользователя B:
%s

Предложи оптимальное слияние этих изменений. Объедини обе версии так, чтобы:
1. Сохранить все полезные изменения из обеих версий
2. Разрешить конфликты наиболее логичным способом
3. Верни только итоговый код без пояснений
`, code1, code2)
	return c.generate(prompt)
}
