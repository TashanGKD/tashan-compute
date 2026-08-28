package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id"`
	Fields    map[string]string `json:"fields,omitempty"`
}

func WriteError(writer http.ResponseWriter, status int, code, message, requestID string, _ error) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(ErrorResponse{Error: ErrorBody{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}})
}

func DecodeJSON(writer http.ResponseWriter, request *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		WriteError(writer, http.StatusBadRequest, "request.invalid_json", "request body is not valid JSON", request.Header.Get("X-Request-ID"), err)
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		WriteError(writer, http.StatusBadRequest, "request.invalid_json", "request body is not valid JSON", request.Header.Get("X-Request-ID"), err)
		return err
	}
	return nil
}
