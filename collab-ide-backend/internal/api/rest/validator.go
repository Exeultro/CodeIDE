package rest

import (
	"regexp"
	"strings"
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ValidateUsername(username string) []ValidationError {
	var errors []ValidationError

	if username == "" {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Имя пользователя обязательно",
		})
	} else if len(username) < 3 {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Имя пользователя должно содержать минимум 3 символа",
		})
	} else if len(username) > 50 {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Имя пользователя не должно превышать 50 символов",
		})
	} else if !regexp.MustCompile(`^[a-zA-Z0-9_а-яА-Я]+$`).MatchString(username) {
		errors = append(errors, ValidationError{
			Field:   "username",
			Message: "Имя пользователя может содержать только буквы, цифры и символ подчеркивания",
		})
	}

	return errors
}

func ValidatePassword(password string) []ValidationError {
	var errors []ValidationError

	if password == "" {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Пароль обязателен",
		})
	} else if len(password) < 6 {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Пароль должен содержать минимум 6 символов",
		})
	} else if len(password) > 100 {
		errors = append(errors, ValidationError{
			Field:   "password",
			Message: "Пароль не должен превышать 100 символов",
		})
	}

	return errors
}

func ValidateSessionName(name string) []ValidationError {
	var errors []ValidationError

	if name == "" {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Название сессии обязательно",
		})
	} else if len(name) > 100 {
		errors = append(errors, ValidationError{
			Field:   "name",
			Message: "Название сессии не должно превышать 100 символов",
		})
	}

	return errors
}

func ValidateFileName(filename string) []ValidationError {
	var errors []ValidationError

	if filename == "" {
		errors = append(errors, ValidationError{
			Field:   "file_name",
			Message: "Имя файла обязательно",
		})
	}

	dangerous := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, d := range dangerous {
		if strings.Contains(filename, d) {
			errors = append(errors, ValidationError{
				Field:   "file_name",
				Message: "Имя файла содержит недопустимые символы",
			})
			break
		}
	}

	return errors
}
