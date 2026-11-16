package gogemini

import "errors"

var (
	// ErrNoAPIKeys is returned when no Gemini API keys are supplied.
	ErrNoAPIKeys = errors.New("go-gemini: at least one API key must be provided")

	// ErrAllKeysExhausted is returned when all rotating API keys are cooling down from 429 rate limits.
	ErrAllKeysExhausted = errors.New("go-gemini: all API keys are currently rate-limited (cooling down)")

	// ErrEmptyPrompt is returned when prompt is empty.
	ErrEmptyPrompt = errors.New("go-gemini: prompt cannot be empty")

	// ErrEmptyPayload is returned when image or media payload is empty.
	ErrEmptyPayload = errors.New("go-gemini: payload cannot be empty")
	ErrModelFailed = errors.New("go-gemini: all Gemini model attempts failed")

	// ErrJSONExtractionFailed is returned when model output does not contain valid parseable JSON.
	ErrJSONExtractionFailed = errors.New("go-gemini: failed to extract valid JSON from Gemini output")
)
