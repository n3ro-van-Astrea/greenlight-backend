package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

type Runtime int32

func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)

	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// Мы ожидаем, что входящее значение JSON будет строкой в формате
	// "<runtime> mins", и первое, что нам нужно сделать — удалить окружающие
	// двойные кавычки из этой строки. Если мы не можем снять кавычки (unquote),
	// то возвращаем ошибку ErrInvalidRuntimeFormat.
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// Разделяем строку, чтобы изолировать часть, содержащую число.
	parts := strings.Split(unquotedJSONValue, " ")

	// Проверяем части строки на корректность, чтобы убедиться, что она соответствовала
	// ожидаемому формату. Если это не так, снова возвращаем ошибку ErrInvalidRuntimeFormat.
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	// В противном случае парсим строку, содержащую число, в тип int32.
	// Опять же, если произошла ошибка, возвращаем ErrInvalidRuntimeFormat.
	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// Преобразуем int32 в тип Runtime и присваиваем его получателю. Обратите внимание,
	// что мы используем оператор * для разыменования получателя (который является
	// указателем на тип Runtime), чтобы установить базовое значение по указателю.
	*r = Runtime(i)
	return nil
}
