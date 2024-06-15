package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"net/url"
)

var EmptyError = errors.New("Emtpy value")
var PermissionDeniedError = errors.New("Permission denied")
var InvalidTokenError = errors.New("Invalid token")
var InternalError = errors.New("Internal error")

type NoSuchKeyError string
type InvalidValueError string
type AlreadyExistsError string

type VersionMismatchError struct {
	Expected int
	Actual   int
}

func (e NoSuchKeyError) Error() string {
	return fmt.Sprintf("No such key: %s", string(e))
}

func (e VersionMismatchError) Error() string {
	return fmt.Sprintf("Version mismatch: expected %v but got %v", e.Expected, e.Actual)
}

func (e InvalidValueError) Error() string {
	return fmt.Sprintf("Invalid value %s", string(e))
}

func (e AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists", string(e))
}

func OkOrDie(err error) {
	if err == nil {
		return
	}
	slog.Error("Fatal error", "error", err)
	panic(err)
}

// GetIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
func GetIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func ErrorResponse(w http.ResponseWriter, code int, message string) {
	JsonResponse(w, code, map[string]string{"error": message})
}

func JsonResponse(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	OkOrDie(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func ValidUrl(raw string) bool {
	_, err := url.ParseRequestURI(raw)
	return err == nil
}

func Hash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}
