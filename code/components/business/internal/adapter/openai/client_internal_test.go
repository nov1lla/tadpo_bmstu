package openai

import "testing"

func TestParseJSONFromContent(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		expectErr bool
		assert    func(t *testing.T, move aiMoveResponse)
	}{
		{
			name:    "plain json",
			content: `{"piece_id":"p1","number":1,"start_row":2,"start_col":3,"path":[{"row":3,"col":4}]}`,
			assert: func(t *testing.T, move aiMoveResponse) {
				if len(move.Path) != 1 {
					t.Fatalf("expected path length 1, got %d", len(move.Path))
				}
				if move.Path[0].Row != 3 || move.Path[0].Col != 4 {
					t.Fatalf("unexpected path element: %+v", move.Path[0])
				}
			},
		},
		{
			name:    "markdown fenced",
			content: "Sure, here's the move:\n```json\n{\n  \"piece_id\": \"p2\",\n  \"number\": 2,\n  \"start_row\": 5,\n  \"start_col\": 4,\n  \"end_row\": 7,\n  \"end_col\": 6\n}\n```",
			assert: func(t *testing.T, move aiMoveResponse) {
				if move.EndRow != 7 || move.EndCol != 6 {
					t.Fatalf("unexpected end position: (%d,%d)", move.EndRow, move.EndCol)
				}
				if len(move.Path) != 0 {
					t.Fatalf("expected empty path, got %d", len(move.Path))
				}
			},
		},
		{
			name:      "no json",
			content:   "no move available",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			move, err := parseJSONFromContent(tc.content)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.assert != nil {
				tc.assert(t, move)
			}
		})
	}
}
