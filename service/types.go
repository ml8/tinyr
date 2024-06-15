package service

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ml8/tinyr/service/util"
)

var IsLetter = regexp.MustCompile(`^[0-9a-zA-Z_\-]+$`).MatchString

type LogEntry struct {
	Host      string    `json:"Host"`
	Timestamp time.Time `json:"Timestamp"`
}

type ListEntry struct {
	Short string `json:"Short"`
	Long  string `json:"Long"`
	Hits  int    `json:"Hits"`
}

type CreateRequest struct {
	Short string `json:"Short"`
	Long  string `json:"Long"`
}
type CreateResponse struct {
}

type DeleteRequest struct {
	Short string `json:"Short"`
}
type DeleteResponse struct {
}

func Parse[T any](r *http.Request, v T) (err error) {
	dec := json.NewDecoder(r.Body)
	err = dec.Decode(v)
	return
}

func ValidShort(short string) (ok bool) {
	return IsLetter(short)
}

func ValidUrl(url string) (err error) {
	if url == "" {
		err = util.EmptyError
	} else if !util.ValidUrl(url) {
		err = util.InvalidValueError(url)
	}
	return
}

func httpify(s string) string {
	if !strings.HasPrefix(s, "https://") && !strings.HasPrefix(s, "http://") {
		return "http://" + s
	}
	return s
}
