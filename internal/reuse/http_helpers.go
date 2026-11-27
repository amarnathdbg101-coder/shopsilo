package reuse

import (
	"encoding/json"
	"net/http"
)

// DecodeJSON decodes a JSON request body into T, writing a 400 Bad Request on failure.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return req, false
	}
	return req, true
}

// DecodeAndValidate decodes a JSON request body into T and validates struct tags,
// writing a 400 Bad Request on decode or validation failure.
func DecodeAndValidate[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	req, ok := DecodeJSON[T](w, r)
	if !ok {
		return req, false
	}
	if err := ValidateStruct(&req); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return req, false
	}
	return req, true
}
