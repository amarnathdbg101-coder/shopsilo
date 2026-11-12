package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
)

const assistantTestConversationID = "7ebdbb37-37d1-4f20-a2ea-12c621a80424"

type assistantTestStore struct {
	conversation *dto.MerchantAssistantConversation
	messages     []dto.MerchantAssistantMessage
	roles        []string
}

func (s *assistantTestStore) CreateConversation(_ context.Context, _, _, title string) (*dto.MerchantAssistantConversation, error) {
	s.conversation = &dto.MerchantAssistantConversation{ID: assistantTestConversationID, Title: title}
	return s.conversation, nil
}

func (s *assistantTestStore) GetConversation(_ context.Context, id, _, _ string) (*dto.MerchantAssistantConversation, error) {
	if s.conversation == nil || s.conversation.ID != id {
		return nil, repository.ErrAIAssistantConversationNotFound
	}
	return s.conversation, nil
}

func (s *assistantTestStore) ListConversations(context.Context, string, string, int) ([]dto.MerchantAssistantConversation, error) {
	return nil, nil
}

func (s *assistantTestStore) ListMessages(context.Context, string, int) ([]dto.MerchantAssistantMessage, error) {
	return append([]dto.MerchantAssistantMessage(nil), s.messages...), nil
}

func (s *assistantTestStore) AddMessage(_ context.Context, _, role, content string, _, _ json.RawMessage) error {
	s.roles = append(s.roles, role)
	s.messages = append(s.messages, dto.MerchantAssistantMessage{
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	})
	return nil
}

type assistantTestShop struct {
	called int
}

func (s *assistantTestShop) GetMyShop(context.Context, string) (*model.Shop, error) {
	s.called++
	return &model.Shop{ID: "shop-1"}, nil
}

type assistantTestTools struct {
	userID string
	names  []string
}

func (t *assistantTestTools) Execute(_ context.Context, userID, name string, _ json.RawMessage) (any, error) {
	t.userID = userID
	t.names = append(t.names, name)
	return map[string]any{"today_sales_amount": 1250.0}, nil
}

type assistantTestModel struct {
	responses []MerchantAssistantModelResponse
	turns     [][]MerchantAssistantTurn
}

func (m *assistantTestModel) Generate(_ context.Context, _ string, turns []MerchantAssistantTurn, _ []MerchantAssistantTool) (MerchantAssistantModelResponse, error) {
	m.turns = append(m.turns, append([]MerchantAssistantTurn(nil), turns...))
	if len(m.responses) == 0 {
		return MerchantAssistantModelResponse{}, errors.New("unexpected model call")
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return response, nil
}

func TestMerchantAssistantChatExecutesReadToolAndPersistsTurn(t *testing.T) {
	store := &assistantTestStore{}
	shop := &assistantTestShop{}
	tools := &assistantTestTools{}
	modelClient := &assistantTestModel{responses: []MerchantAssistantModelResponse{
		{
			FunctionCalls: []MerchantAssistantFunctionCall{{
				Name: "get_shop_overview",
				Args: json.RawMessage(`{}`),
			}},
		},
		{Text: "Aaj ki sales ₹1,250 hain."},
	}}
	service := NewMerchantAssistantService(store, shop, tools, modelClient)

	response, err := service.Chat(context.Background(), "owner-1", dto.MerchantAssistantChatRequest{
		Prompt: "Aaj ki sales kitni hain?",
	})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if response.ConversationID != assistantTestConversationID {
		t.Fatalf("unexpected conversation ID: %q", response.ConversationID)
	}
	if response.Text != "Aaj ki sales ₹1,250 hain." {
		t.Fatalf("unexpected response text: %q", response.Text)
	}
	if len(response.ToolsUsed) != 1 || response.ToolsUsed[0] != "get_shop_overview" {
		t.Fatalf("unexpected tools used: %+v", response.ToolsUsed)
	}
	if shop.called != 1 || tools.userID != "owner-1" {
		t.Fatalf("shop context was not resolved from authenticated user: shop calls=%d, tool user=%q", shop.called, tools.userID)
	}
	if len(store.roles) != 2 || store.roles[0] != "user" || store.roles[1] != "assistant" {
		t.Fatalf("unexpected persisted message roles: %+v", store.roles)
	}
	if len(modelClient.turns) != 2 || len(modelClient.turns[1]) != 3 {
		t.Fatalf("tool call/result were not passed back to the model: %+v", modelClient.turns)
	}
	lastTurn := modelClient.turns[1][2]
	if lastTurn.Role != "tool" || len(lastTurn.FunctionResponses) != 1 {
		t.Fatalf("missing tool response in follow-up: %+v", lastTurn)
	}
}

func TestMerchantAssistantChatDoesNotAccessForeignConversation(t *testing.T) {
	store := &assistantTestStore{}
	shop := &assistantTestShop{}
	modelClient := &assistantTestModel{}
	service := NewMerchantAssistantService(store, shop, &assistantTestTools{}, modelClient)

	_, err := service.Chat(context.Background(), "owner-1", dto.MerchantAssistantChatRequest{
		ConversationID: assistantTestConversationID,
		Prompt:         "show sales",
	})
	if !errors.Is(err, ErrMerchantAssistantConversation) {
		t.Fatalf("expected conversation not found, got %v", err)
	}
	if len(modelClient.turns) != 0 || len(store.roles) != 0 {
		t.Fatal("foreign conversation triggered model or message writes")
	}
}

func TestMerchantAssistantChatRejectsUnregisteredModelTool(t *testing.T) {
	store := &assistantTestStore{}
	tools := &assistantTestTools{}
	modelClient := &assistantTestModel{responses: []MerchantAssistantModelResponse{{
		FunctionCalls: []MerchantAssistantFunctionCall{{
			Name: "delete_shop",
			Args: json.RawMessage(`{}`),
		}},
	}}}
	service := NewMerchantAssistantService(store, &assistantTestShop{}, tools, modelClient)

	_, err := service.Chat(context.Background(), "owner-1", dto.MerchantAssistantChatRequest{
		Prompt: "delete my shop",
	})
	if !errors.Is(err, ErrMerchantAssistantTool) {
		t.Fatalf("expected unregistered tool to fail, got %v", err)
	}
	if len(tools.names) != 0 {
		t.Fatalf("unregistered tool reached the executor: %+v", tools.names)
	}
}

func TestDecodeAssistantToolArgsRejectsUnexpectedAndNonObjectInput(t *testing.T) {
	for _, raw := range []string{
		`{"query":"atta","extra":"field"}`,
		`null`,
		`{"query":"atta"} {"query":"rice"}`,
	} {
		var args struct {
			Query string `json:"query"`
		}
		if err := decodeAssistantToolArgs(json.RawMessage(raw), &args); err == nil {
			t.Errorf("expected tool arguments %s to be rejected", raw)
		}
	}
}

func TestBuildGeminiAssistantPayloadEncodesFunctionResults(t *testing.T) {
	payload, err := buildGeminiAssistantPayload("system", []MerchantAssistantTurn{
		{Role: "user", Text: "show sales"},
		{
			Role: "model",
			FunctionCalls: []MerchantAssistantFunctionCall{{
				Name: "get_shop_overview",
				Args: json.RawMessage(`{}`),
			}},
		},
		{
			Role: "tool",
			FunctionResponses: []MerchantAssistantFunctionResponse{{
				Name:   "get_shop_overview",
				Result: json.RawMessage(`{"today_sales_amount":1250}`),
			}},
		},
	}, merchantAssistantToolDefinitions)
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}
	if payload.ToolConfig.FunctionCallingConfig.Mode != "AUTO" {
		t.Fatalf("unexpected function calling mode: %q", payload.ToolConfig.FunctionCallingConfig.Mode)
	}
	if len(payload.Contents) != 3 || payload.Contents[2].Role != "user" {
		t.Fatalf("tool response was not represented as a user function-response turn: %+v", payload.Contents)
	}
	if payload.Contents[1].Parts[0].FunctionCall == nil || payload.Contents[2].Parts[0].FunctionResponse == nil {
		t.Fatal("function call/response parts are missing")
	}
}

func TestMerchantAssistantToolExecutorRejectsUnregisteredTools(t *testing.T) {
	executor := NewMerchantAssistantToolExecutor(nil, nil, nil)
	if _, err := executor.Execute(context.Background(), "owner-1", "delete_shop", json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected an unregistered tool to be rejected")
	}
}
