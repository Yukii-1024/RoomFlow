// Package httpx contains the shared HTTP envelope, error model, request
// decoding and parameter helpers used by every handler.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Envelope is the single response shape for every endpoint.
type Envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// HTTPError carries the HTTP status, machine-readable code and human message.
type HTTPError struct {
	Status  int
	Code    string
	Message string
	Cause   error
}

func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error { return e.Cause }

func NewError(status int, code, message string) *HTTPError {
	return &HTTPError{Status: status, Code: code, Message: message}
}

func BadRequest(message string) *HTTPError {
	return NewError(http.StatusBadRequest, "INVALID_REQUEST", message)
}
func Unauthorized(message string) *HTTPError {
	return NewError(http.StatusUnauthorized, "UNAUTHORIZED", message)
}
func Forbidden(message string) *HTTPError {
	return NewError(http.StatusForbidden, "FORBIDDEN", message)
}
func NotFound(message string) *HTTPError { return NewError(http.StatusNotFound, "NOT_FOUND", message) }

func Conflict(code, message string) *HTTPError {
	return NewError(http.StatusConflict, code, message)
}

// Internal wraps an unexpected error and hides its details from clients.
func Internal(err error) *HTTPError {
	return &HTTPError{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_ERROR",
		Message: "服务器内部错误",
		Cause:   err,
	}
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Code: "OK", Message: "success", Data: data})
}

func WriteError(w http.ResponseWriter, err error) {
	var he *HTTPError
	if !errors.As(err, &he) {
		he = Internal(err)
	}
	if he.Status >= http.StatusInternalServerError {
		slog.Error("request failed", "error", he.Error())
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(he.Status)
	_ = json.NewEncoder(w).Encode(Envelope{Code: he.Code, Message: he.Message, Data: nil})
}

// Decode strictly decodes a single JSON object from the request body.
func Decode(r *http.Request, dst any) error {
	if r.Body == nil {
		return BadRequest("请求体不能为空")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return BadRequest("请求体格式错误: " + err.Error())
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return BadRequest("请求体只能包含一个 JSON 对象")
	}
	return nil
}

// DecodeOptional tolerates an empty body (used by action endpoints whose body
// is optional, e.g. check-in).
func DecodeOptional(r *http.Request, dst any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	return Decode(r, dst)
}

func PathInt(r *http.Request, name string) (int64, error) {
	raw := r.PathValue(name)
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0, BadRequest("路径参数 " + name + " 无效")
	}
	return n, nil
}

func QueryInt(r *http.Request, name string, fallback int) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, BadRequest("查询参数 " + name + " 必须是整数")
	}
	return n, nil
}

func QueryInt64(r *http.Request, name string) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return 0, BadRequest("查询参数 " + name + " 不能为空")
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0, BadRequest("查询参数 " + name + " 无效")
	}
	return n, nil
}

// Pagination normalizes page and page_size (1..100, default 20).
func Pagination(r *http.Request) (page, pageSize int, err error) {
	page, err = QueryInt(r, "page", 1)
	if err != nil {
		return 0, 0, err
	}
	pageSize, err = QueryInt(r, "page_size", 20)
	if err != nil {
		return 0, 0, err
	}
	if page < 1 {
		return 0, 0, BadRequest("page 必须大于等于 1")
	}
	if pageSize < 1 || pageSize > 100 {
		return 0, 0, BadRequest("page_size 必须在 1 到 100 之间")
	}
	return page, pageSize, nil
}

const DateLayout = "2006-01-02"

func ParseDate(raw string) (time.Time, error) {
	d, err := time.Parse(DateLayout, raw)
	if err != nil {
		return time.Time{}, BadRequest("日期格式必须为 YYYY-MM-DD")
	}
	return d, nil
}

func QueryDate(r *http.Request, name string) (time.Time, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return time.Time{}, BadRequest("查询参数 " + name + " 不能为空")
	}
	return ParseDate(raw)
}

func FormatDate(t time.Time) string { return t.Format(DateLayout) }

// Nights returns whole days between two date-only values.
func Nights(checkIn, checkOut time.Time) int {
	return int(checkOut.Sub(checkIn).Hours() / 24)
}

// Today returns the current local calendar date at UTC midnight so it can be
// compared with parsed YYYY-MM-DD values.
func Today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// Principal is the authenticated caller derived from the JWT.
type Principal struct {
	UserID int64
	Role   string
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

func MustPrincipal(ctx context.Context) Principal {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		panic("httpx: missing principal in context")
	}
	return p
}
