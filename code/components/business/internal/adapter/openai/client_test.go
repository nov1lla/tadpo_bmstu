package openai_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ppo/business/internal/adapter/openai"
	"ppo/sdk/domain"
	"ppo/sdk/port"
)

func TestClientSuggestMoveSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices": [{"message": {"content": "{\"piece_id\":\"piece-1\",\"number\":2,\"start_row\":2,\"start_col\":3,\"path\":[{\"row\":3,\"col\":4}]}"}}]}`))
	}))
	defer server.Close()

	httpClient := server.Client()
	httpClient.Timeout = time.Second

	client, err := openai.NewClient("test-key",
		openai.WithHTTPClient(httpClient),
		openai.WithBaseURL(server.URL),
		openai.WithModel("test-model"),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	board := domain.NewBoardState(8)
	board.Pieces["piece-1"] = domain.BoardPiece{
		ID:       "piece-1",
		Position: domain.Position{Row: 2, Col: 3},
		Color:    domain.PlayerColorDark,
		Kind:     domain.PieceKindMan,
	}
	req := port.OpponentMoveRequest{
		Game:       domain.Game{ID: "game-1"},
		Board:      board,
		Difficulty: domain.DifficultyMedium,
	}

	move, err := client.SuggestMove(context.Background(), req)
	if err != nil {
		t.Fatalf("suggest move: %v", err)
	}
	if move.GameID != req.Game.ID {
		t.Fatalf("unexpected game id: %s", move.GameID)
	}
	if move.PieceID != "piece-1" || move.EndPosition.Row != 3 {
		t.Fatalf("unexpected move: %+v", move)
	}
	if len(move.Trajectory) != 1 {
		t.Fatalf("expected trajectory of length 1, got %d", len(move.Trajectory))
	}
}

func TestClientSuggestMoveAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid request"}`))
	}))
	defer server.Close()

	client, err := openai.NewClient("test-key",
		openai.WithHTTPClient(server.Client()),
		openai.WithBaseURL(server.URL),
		openai.WithModel("test-model"),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	req := port.OpponentMoveRequest{Game: domain.Game{ID: "game-1"}}
	_, err = client.SuggestMove(context.Background(), req)
	var apiErr openai.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.Status != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", apiErr.Status)
	}
}
