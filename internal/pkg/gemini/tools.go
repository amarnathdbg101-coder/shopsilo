package gogemini

// FunctionDeclaration defines a function callable by Gemini models.
type FunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// Tool groups function declarations to enable Gemini function calling.
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"function_declarations"`
}

// FunctionCall represents a tool invocation requested by Gemini.
type FunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// FunctionResponse returns an executed tool's result to Gemini.
type FunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// SystemInstruction specifies persistent persona and behavioral directives for Gemini.
type SystemInstruction struct {
	Parts []Part `json:"parts"`
}

// ChatRequest configures a multi-turn dialogue with optional system instruction and tools.
type ChatRequest struct {
	SystemInstruction string
	Contents          []Content
	Tools             []Tool
	Temperature       *float64
	MaxOutputTokens   int
}

// ChatResponse holds the model's text response and/or requested tool function calls.
type ChatResponse struct {
	Text          string         `json:"text"`
	FunctionCalls []FunctionCall `json:"function_calls,omitempty"`
}
