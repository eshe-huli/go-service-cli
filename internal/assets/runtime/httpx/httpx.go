// Package httpx implements the one supported JSON transport boundary.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"gsvc.local/cli/internal/assets/runtime/fault"
)

const MaxBodyBytes = 1 << 20

// DecodeJSON requires exactly one object, rejects duplicate/unknown keys, nulls
// for required fields and bodies over the limit. Required bool/int fields may
// legitimately contain false/zero; presence is validated independently.
func DecodeJSON(w http.ResponseWriter, r *http.Request, out any, required ...string) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return fault.New(fault.Invalid, "Content-Type must be application/json")
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	if err != nil {
		return fault.New(fault.Invalid, "body is unreadable or exceeds 1 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(body))
	tok, err := d.Token()
	if err != nil || tok != json.Delim('{') {
		return fault.New(fault.Invalid, "body must be a JSON object")
	}
	allowed := map[string]bool{}
	for _, key := range required {
		allowed[key] = true
	}
	present := map[string]bool{}
	for d.More() {
		tok, err = d.Token()
		if err != nil {
			return fault.New(fault.Invalid, "invalid JSON object")
		}
		key, ok := tok.(string)
		if !ok || present[key] {
			return fault.New(fault.Invalid, "duplicate JSON field")
		}
		if !allowed[key] {
			return fault.New(fault.Invalid, "unknown JSON field: "+key)
		}
		var raw json.RawMessage
		if err = d.Decode(&raw); err != nil {
			return fault.New(fault.Invalid, "invalid JSON value")
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fault.New(fault.Invalid, "null fields are not supported")
		}
		present[key] = true
	}
	if _, err = d.Token(); err != nil {
		return fault.New(fault.Invalid, "invalid JSON object")
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return fault.New(fault.Invalid, "body must contain exactly one JSON object")
	}
	for _, key := range required {
		if !present[key] {
			return fault.New(fault.Invalid, key+" is required")
		}
	}
	d = json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err = d.Decode(out); err != nil {
		return fault.New(fault.Invalid, "body contains unknown fields or invalid field types")
	}
	return nil
}
func Query(r *http.Request, allowed ...string) (url.Values, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, fault.New(fault.Invalid, "invalid query string")
	}
	keys := map[string]bool{}
	for _, k := range allowed {
		keys[k] = true
	}
	for k, v := range q {
		if !keys[k] || len(v) != 1 {
			return nil, fault.New(fault.Invalid, "unknown or repeated query field: "+k)
		}
	}
	return q, nil
}
func String(q url.Values, key string) (string, error) {
	v, ok := q[key]
	if !ok || len(v) != 1 || strings.TrimSpace(v[0]) == "" {
		return "", fault.New(fault.Invalid, key+" is required")
	}
	return v[0], nil
}
func Int64(q url.Values, key string) (int64, error) {
	s, err := String(q, key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fault.New(fault.Invalid, key+" must be an int64")
	}
	return v, nil
}
func Bool(q url.Values, key string) (bool, error) {
	s, err := String(q, key)
	if err != nil {
		return false, err
	}
	if s != "true" && s != "false" {
		return false, fault.New(fault.Invalid, key+" must be true or false")
	}
	return s == "true", nil
}
func JSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		Failure(w, errors.New("response serialization failed"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(append(b, '\n'))
}
func Failure(w http.ResponseWriter, err error) {
	status, code, message := http.StatusInternalServerError, "internal_error", "internal error"
	var f *fault.Error
	if errors.As(err, &f) {
		switch f.Kind {
		case fault.Invalid:
			status = 400
		case fault.NotFound:
			status = 404
		case fault.Conflict:
			status = 409
		case fault.Forbidden:
			status = 403
		case fault.Unauthenticated:
			status = 401
		case fault.NotImplemented:
			status = 501
		}
		if status != 500 {
			code = string(f.Kind)
			message = f.Message
		}
	} else if errors.Is(err, context.DeadlineExceeded) {
		status = 504
		code = "deadline_exceeded"
		message = "operation timed out"
	} else if errors.Is(err, context.Canceled) {
		status = 408
		code = "request_canceled"
		message = "request was canceled"
	}
	if status == 500 {
		slog.Error("request failed", "error", err)
	}
	JSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

// Recover is a final boundary for unexpected panics. Normal failures are errors.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				slog.Error("request panic", "panic", v)
				Failure(w, errors.New("unexpected panic"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
