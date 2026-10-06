package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type envelope map[string]any

func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parametr")
	}

	return id, nil
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	// Добавляем заголовок "Content-Type: application/json", затем пишем
	// статус-код и JSON-ответ.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {

	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)

	if err != nil {
		// Если во время декодирования произошла ошибка, начинаем сортировку...
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		var maxBytesError *http.MaxBytesError

		switch {
		// Используем функцию errors.As(), чтобы проверить, имеет ли ошибка тип *json.SyntaxError.
		// Если да, возвращаем понятное сообщение об ошибке с указанием позиции проблемы.
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		// В некоторых случаях Decode() также может возвращать io.ErrUnexpectedEOF
		// при синтаксических ошибках в JSON. Проверяем это с помощью errors.Is()
		// и возвращаем общее сообщение. Открытый issue по этому поводу находится здесь:
		// https://github.com/golang/go/issues/25956.
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		// Аналогично перехватываем ошибки *json.UnmarshalTypeError. Они возникают, когда
		// тип значения в JSON не соответствует целевому типу. Если ошибка связана с
		// конкретным полем, включаем его в сообщение, чтобы клиенту было проще отлаживать.
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		// Ошибка io.EOF возвращается методом Decode(), если тело запроса пустое.
		// Проверяем её через errors.Is() и возвращаем понятное сообщение.
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown kew %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

		// Ошибка json.InvalidUnmarshalError возвращается, если мы передали в Decode()
		// что-то помимо ненулевого указателя (non-nil pointer). Мы перехватываем её и
		// вызываем panic, вместо того чтобы возвращать ошибку в хендлер. В конце этой
		// главы мы поговорим о паниках против возврата ошибок и обсудим, почему в этой
		// ситуации panic уместен.
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		// Во всех остальных случаях возвращаем ошибку как есть.
		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}
