package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"ppo/sdk/domain"
	"ppo/sdk/port"
)

var (
	ErrEmptyResponse = errors.New("openai: empty completion result")
	ErrMalformedJSON = errors.New("openai: malformed json in completion")
)

type APIError struct {
	Status int
	Body   string
}

func (e APIError) Error() string {
	return fmt.Sprintf("openai api error: status=%d body=%s", e.Status, e.Body)
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	model      string
	apiKey     string
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithModel(model string) Option {
	return func(c *Client) {
		if model != "" {
			c.model = model
		}
	}
}

func WithBaseURL(url string) Option {
	return func(c *Client) {
		if url != "" {
			c.baseURL = url
		}
	}
}

func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("openai api key is empty")
	}
	client := &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		apiKey:     apiKey,
	}
	for _, opt := range opts {
		opt(client)
	}
	if client.baseURL == "" {
		return nil, errors.New("openai base url not configured")
	}
	if client.model == "" {
		return nil, errors.New("openai model not configured")
	}
	return client, nil
}

var _ port.OpponentMoveProvider = (*Client)(nil)

func (c *Client) SuggestMove(ctx context.Context, req port.OpponentMoveRequest) (domain.Move, error) {
	payload, err := c.buildRequest(req)
	if err != nil {
		return domain.Move{}, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return domain.Move{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return domain.Move{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return domain.Move{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return domain.Move{}, APIError{Status: resp.StatusCode, Body: string(bodyBytes)}
	}

	var completion completionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return domain.Move{}, err
	}
	if len(completion.Choices) == 0 {
		return domain.Move{}, ErrEmptyResponse
	}

	content := completion.Choices[0].Message.Content
	var move aiMoveResponse
	if err := json.Unmarshal([]byte(content), &move); err != nil {

		move, err = parseJSONFromContent(content)
		if err != nil {
			return domain.Move{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
		}
	}

	if err := move.validate(); err != nil {
		return domain.Move{}, err
	}

	trajectory := move.toTrajectory()
	domainMove, err := domain.NewMove(
		domain.MoveID(uuid.NewString()),
		req.Game.ID,
		domain.PieceID(move.PieceID),
		domain.MoveNumber(move.Number),
		domain.Position{Row: domain.Coordinate(move.StartRow), Col: domain.Coordinate(move.StartCol)},
		trajectory,
		domain.NewTimestamp(time.Now()),
	)
	if err != nil {
		return domain.Move{}, err
	}
	return domainMove, nil
}

func (c *Client) buildRequest(req port.OpponentMoveRequest) (chatCompletionRequest, error) {
	boardJSON := describeBoard(req.Board)
	opponentColor := colorLabel(req.Game.PlayerColor.Opponent())
	playerColor := colorLabel(req.Game.PlayerColor)
	feedback := formatFeedback(req.Feedback)
	prompt := fmt.Sprintf(`You are a %s checkers AI. Analyze the current board snapshot (no history is provided) and return ONE legal move for your side.

Rules summary:
- English checkers on an %d×%d board. Rows are indexed from 0 at the top (increase downward); columns from 0 at the left (increase to the right). Square (0,0) is the top-left corner.
- Play at maximum strength for %s: inspect every legal move, prefer capturing sequences, and pick the option that maximizes material or positional advantage.
- %s pieces move toward larger row numbers (downward). %s pieces move toward smaller row numbers (upward).
- Regular men move diagonally forward by one square; kings move diagonally in both directions.
- Regular men capture diagonally forward only. Kings may capture diagonally in any direction.
- Captures are made by jumping over an adjacent opponent piece onto the empty square immediately beyond it; multi-jumps are allowed whenever each landing square is empty.
- Captures are MANDATORY: if any capture is available for your side anywhere on the board, you MUST return a capturing move.
- Multi-capture continuation is MANDATORY: if you capture and from the new landing square the same piece can capture again, you MUST continue capturing in the same move until no further captures are possible.
- The landing squares after each jump must be empty before the move is played.
- The "path" must list ONLY landing squares after each step/jump (do NOT include the starting square or any jumped-over pieces).
- NEVER move onto a square occupied by your own piece; rely on the JSON board listing below.
%s

Return ONLY a JSON object with fields:
- piece_id (string)
- number (int) – sequential move number
- start_row (int)
- start_col (int)
- end_row (int)
- end_col (int)
- path (array of objects {row:int,col:int}) listing the landing squares in order (each element is a square where the moving piece stops after a step or capture).

Board state (JSON map keyed by piece_id):
%s
`, opponentColor, int(req.Board.Size), int(req.Board.Size), opponentColor, opponentColor, playerColor, feedback, boardJSON)

	log.Printf("openai model=%s", c.model)

	return chatCompletionRequest{
		Model:       c.model,
		Temperature: 0.1,
		ResponseFormat: &responseFormat{
			Type: "json_object",
		},
		Messages: []chatMessage{
			{Role: "system", Content: "You are an assistant that outputs valid JSON for checkers moves."},
			{Role: "user", Content: prompt},
		},
	}, nil
}

type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Temperature    float64         `json:"temperature,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Messages       []chatMessage   `json:"messages"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type aiMoveResponse struct {
	PieceID  string        `json:"piece_id"`
	Number   int           `json:"number"`
	StartRow int           `json:"start_row"`
	StartCol int           `json:"start_col"`
	EndRow   int           `json:"end_row"`
	EndCol   int           `json:"end_col"`
	Path     []pathElement `json:"path"`
}

type pathElement struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

func (m aiMoveResponse) validate() error {
	if m.PieceID == "" {
		return errors.New("piece_id missing")
	}
	if m.Number <= 0 {
		return errors.New("number must be positive")
	}
	return nil
}

func (m aiMoveResponse) toTrajectory() []domain.Position {
	trajectory := make([]domain.Position, 0, len(m.Path))
	for _, step := range m.Path {
		trajectory = append(trajectory, domain.Position{Row: domain.Coordinate(step.Row), Col: domain.Coordinate(step.Col)})
	}
	if len(trajectory) == 0 {
		trajectory = append(trajectory, domain.Position{Row: domain.Coordinate(m.EndRow), Col: domain.Coordinate(m.EndCol)})
	}
	return trajectory
}

func describeBoard(state domain.BoardState) string {
	size := state.Size
	if size == 0 {
		size = domain.DefaultBoardSize
	}
	type summary struct {
		Color string `json:"color"`
		Kind  string `json:"kind"`
		Row   int    `json:"row"`
		Col   int    `json:"col"`
	}
	pieces := make(map[string]summary, len(state.Pieces))
	for id, piece := range state.Pieces {
		pieces[string(id)] = summary{
			Color: colorLabel(piece.Color),
			Kind:  string(piece.Kind),
			Row:   int(piece.Position.Row),
			Col:   int(piece.Position.Col),
		}
	}
	board := struct {
		Size   int                `json:"size"`
		Pieces map[string]summary `json:"pieces"`
	}{
		Size:   int(size),
		Pieces: pieces,
	}
	bytes, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\"error\":%q}", err.Error())
	}
	return string(bytes)
}

func formatFeedback(feedback []string) string {
	if len(feedback) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("\nPrevious invalid attempts (avoid repeating):\n")
	for idx, item := range feedback {
		builder.WriteString(fmt.Sprintf("%d) %s\n", idx+1, item))
	}
	return builder.String()
}

func colorLabel(color domain.PlayerColor) string {
	if color == domain.PlayerColorLight {
		return "white"
	}
	return "black"
}

func parseJSONFromContent(content string) (aiMoveResponse, error) {
	trimmed := strings.TrimSpace(content)
	var move aiMoveResponse

	if err := json.Unmarshal([]byte(trimmed), &move); err == nil {
		return move, nil
	}

	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```")
		trimmed = strings.TrimSpace(trimmed)
		if idx := strings.IndexByte(trimmed, '\n'); idx != -1 {

			trimmed = strings.TrimSpace(trimmed[idx+1:])
		}
		if endFence := strings.LastIndex(trimmed, "```"); endFence != -1 {
			trimmed = trimmed[:endFence]
		}
		if err := json.Unmarshal([]byte(trimmed), &move); err == nil {
			return move, nil
		}
	}

	depth := 0
	start := -1
	var lastErr error
	for i, r := range content {
		switch r {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 && start != -1 {
					fragment := content[start : i+1]
					if err := json.Unmarshal([]byte(fragment), &move); err == nil {
						return move, nil
					} else {
						lastErr = err
					}
				}
			}
		}
	}

	if lastErr != nil {
		return aiMoveResponse{}, lastErr
	}
	return aiMoveResponse{}, errors.New("no json object found in content")
}
