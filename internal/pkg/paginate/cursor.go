package paginate

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidCursor = errors.New("paginate: invalid cursor format")
)

// CursorParams holds parameters for cursor-based (keyset) pagination.
// Infinite scrolling feeds, chat messages, aur massive datasets ke liye use hota hai.
type CursorParams struct {
	Cursor string `json:"cursor"` // Opaque base64 token of the last seen item
	Limit  int    `json:"limit"`  // Number of items to fetch (default: 20, max: 100)
	Order  string `json:"order"`  // "ASC" or "DESC"
}

// CursorMeta holds pagination state returned to mobile/web clients.
type CursorMeta struct {
	NextCursor string `json:"next_cursor,omitempty"` // Pass this into ?cursor= for the next page
	HasMore    bool   `json:"has_more"`              // True if more records are available
	Limit      int    `json:"limit"`                 // Requested limit
}

// CursorFromRequest extracts cursor pagination parameters from HTTP request.
func CursorFromRequest(r *http.Request, defaultLimit int) CursorParams {
	return CursorFromValues(r.URL.Query(), defaultLimit)
}

// CursorFromValues extracts cursor parameters from url.Values map.
func CursorFromValues(values url.Values, defaultLimit int) CursorParams {
	if defaultLimit <= 0 {
		defaultLimit = 20
	}

	limit := defaultLimit
	if val := values.Get("limit"); val != "" {
		if l, err := strconv.Atoi(val); err == nil && l > 0 {
			if l > MaxPerPage {
				limit = MaxPerPage
			} else {
				limit = l
			}
		}
	}

	cursor := strings.TrimSpace(values.Get("cursor"))

	order := OrderDesc
	if val := values.Get("order"); val != "" {
		if strings.EqualFold(strings.TrimSpace(val), OrderAsc) {
			order = OrderAsc
		}
	}

	return CursorParams{
		Cursor: cursor,
		Limit:  limit,
		Order:  order,
	}
}

// EncodeCursor encodes a simple primary key ID into a URL-safe Base64 opaque cursor.
func EncodeCursor(id string) string {
	if id == "" {
		return ""
	}
	return base64.URLEncoding.EncodeToString([]byte(id))
}

// DecodeCursor decodes a URL-safe Base64 cursor back into the underlying ID string.
func DecodeCursor(cursorStr string) (string, error) {
	if cursorStr == "" {
		return "", nil
	}
	bytes, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		return "", ErrInvalidCursor
	}
	return string(bytes), nil
}

// EncodeCompositeCursor encodes both a timestamp and an ID into a composite cursor.
// Time-ordered feeds (e.g. ORDER BY created_at DESC, id DESC) ke liye duplicate timestamps ko handle karta hai.
func EncodeCompositeCursor(t time.Time, id string) string {
	raw := fmt.Sprintf("%d:%s", t.UnixNano(), id)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCompositeCursor parses a composite cursor back into timestamp and ID.
func DecodeCompositeCursor(cursorStr string) (time.Time, string, error) {
	if cursorStr == "" {
		return time.Time{}, "", nil
	}
	bytes, err := base64.URLEncoding.DecodeString(cursorStr)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}

	parts := strings.SplitN(string(bytes), ":", 2)
	if len(parts) != 2 {
		return time.Time{}, "", ErrInvalidCursor
	}

	nano, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}

	return time.Unix(0, nano), parts[1], nil
}
