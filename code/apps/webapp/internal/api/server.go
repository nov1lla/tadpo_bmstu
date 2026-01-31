package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"

	"ppo/sdk/component"
	"ppo/sdk/domain"
	sdkusecase "ppo/sdk/usecase"
)

type Server struct {
	mux       *http.ServeMux
	static    http.Handler
	indexPath string

	users sdkusecase.UserUseCase
	games sdkusecase.GameUseCase
	moves sdkusecase.MoveUseCase
	auth  sdkusecase.AuthUseCase
	anim  sdkusecase.MoveAnimationUseCase

	logCfg httpLogConfig
}

func NewServer(staticDir string, business component.BusinessProvider) (*Server, error) {
	if business == nil {
		return nil, errors.New("business provider is nil")
	}
	absStatic := staticDir
	if !filepath.IsAbs(absStatic) {
		cwd, err := os.Getwd()
		if err == nil {
			absStatic = filepath.Clean(filepath.Join(cwd, staticDir))
		}
	}
	index := filepath.Join(absStatic, "index.html")
	srv := &Server{
		mux:       http.NewServeMux(),
		static:    http.FileServer(http.Dir(absStatic)),
		indexPath: index,
		users:     business.UserUseCase(),
		games:     business.GameUseCase(),
		moves:     business.MoveUseCase(),
		auth:      business.AuthUseCase(),
		anim:      business.MoveAnimationUseCase(),
		logCfg:    loadHTTPLogConfigFromEnv(),
	}
	srv.registerRoutes()
	return srv, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var reqBody *limitedBuffer
	if s.logCfg.logBodies && r.Body != nil {
		reqBody = newLimitedBuffer(s.logCfg.maxBodyBytes)
		r.Body = &teeReadCloser{rc: r.Body, dst: reqBody}
	}

	recorder := &responseRecorder{
		ResponseWriter: w,
		status:         http.StatusOK,
		captureBody:    s.logCfg.logBodies,
		body:           newLimitedBuffer(s.logCfg.maxBodyBytes),
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.mux.ServeHTTP(recorder, r)
	} else {
		s.serveStatic(recorder, r)
	}

	traceID := ""
	if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
		traceID = sc.TraceID().String()
	}

	if s.logCfg.level == "debug" {
		var reqSnippet string
		if reqBody != nil {
			reqSnippet = reqBody.String()
		}
		log.Printf(
			"http %s %s status=%d bytes=%d duration=%s trace_id=%s req=%s resp=%s",
			r.Method,
			r.URL.Path,
			recorder.status,
			recorder.bytes,
			time.Since(start),
			traceID,
			compactJSON(reqSnippet),
			compactJSON(recorder.body.String()),
		)
		return
	}

	log.Printf("http %s %s status=%d bytes=%d duration=%s trace_id=%s",
		r.Method,
		r.URL.Path,
		recorder.status,
		recorder.bytes,
		time.Since(start),
		traceID,
	)
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)
	if path == "/" || path == "." {
		http.ServeFile(w, r, s.indexPath)
		return
	}
	fullPath := filepath.Join(filepath.Dir(s.indexPath), path)
	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		s.static.ServeHTTP(w, r)
		return
	}
	http.ServeFile(w, r, s.indexPath)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/api/users", s.handleUsers)
	s.mux.HandleFunc("/api/users/", s.handleUserByID)
	s.mux.HandleFunc("/api/games", s.handleGames)
	s.mux.HandleFunc("/api/games/", s.handleGameByID)
	s.mux.HandleFunc("/api/auth/register", s.handleRegister)
	s.mux.HandleFunc("/api/auth/login", s.handleLogin)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		return
	}
	var req authRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, err := s.auth.Register(r.Context(), sdkusecase.RegisterCommand{
		Login:    domain.Login(req.Login),
		Password: req.Password,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{User: newUserDTO(user)})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		return
	}
	var req authRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, err := s.auth.Login(r.Context(), sdkusecase.LoginCommand{
		Login:    domain.Login(req.Login),
		Password: req.Password,
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{User: newUserDTO(user)})
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req createUserRequest
		if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		user, err := s.users.Create(r.Context(), sdkusecase.CreateUserCommand{Name: domain.UserName(req.Name)})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, newUserDTO(user))
	default:
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
	}
}

func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if userID == "" {
		writeError(w, http.StatusNotFound, errors.New("user not specified"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		user, err := s.users.Get(r.Context(), domain.UserID(userID))
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, newUserDTO(user))
	case http.MethodPatch:
		var req updateUserRequest
		if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		cmd := sdkusecase.UpdateUserFieldsCommand{ID: domain.UserID(userID)}
		if req.Rating != nil {
			rating := domain.Rating(*req.Rating)
			cmd.Rating = &rating
		}
		if req.WinStreak != nil {
			streak := domain.WinStreak(*req.WinStreak)
			cmd.WinStreak = &streak
		}
		if req.LastGameID != nil {
			gameID := domain.GameID(*req.LastGameID)
			cmd.LastGameID = &gameID
		}
		user, err := s.users.UpdateFields(r.Context(), cmd)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, newUserDTO(user))
	default:
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
	}
}

func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req createGameRequest
		if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		playerColor := domain.PlayerColor(strings.ToLower(req.PlayerColor))
		if playerColor != domain.PlayerColorLight && playerColor != domain.PlayerColorDark {
			playerColor = domain.PlayerColorLight
		}
		game, err := s.games.New(r.Context(), sdkusecase.CreateGameCommand{
			UserID:        domain.UserID(req.UserID),
			PlayerColor:   playerColor,
			IsPlayerFirst: req.IsPlayerFirst,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		board, err := s.games.GetChessboard(r.Context(), game.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		response := gameStateDTO{
			Game:  newGameDTO(game),
			Board: newBoardDTO(board),
			Moves: []moveDTO{},
		}
		writeJSON(w, http.StatusCreated, response)
	case http.MethodGet:
		userID := strings.TrimSpace(r.URL.Query().Get("userId"))
		if userID == "" {
			writeError(w, http.StatusBadRequest, errors.New("userId query parameter required"))
			return
		}
		games, err := s.games.ListByUser(r.Context(), domain.UserID(userID))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, gamesListDTO{Games: newGamesDTO(games)})
	default:
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
	}
}

func (s *Server) handleGameByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/games/")
	if path == "" {
		writeError(w, http.StatusNotFound, errors.New("game not specified"))
		return
	}
	parts := strings.SplitN(path, "/", 2)
	gameID := domain.GameID(parts[0])
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.getGameState(w, r, gameID)
		case http.MethodDelete:
			s.deleteGame(w, r, gameID)
		default:
			writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		}
		return
	}
	switch parts[1] {
	case "board":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
			return
		}
		board, err := s.games.GetChessboard(r.Context(), gameID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, newBoardDTO(board))
	case "moves":
		switch r.Method {
		case http.MethodGet:
			history, err := s.games.GenerateHistory(r.Context(), gameID)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeJSON(w, http.StatusOK, newMovesDTO(history))
		case http.MethodPost:
			s.applyUserMove(w, r, gameID)
		default:
			writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		}
	case "opponent-move":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
			return
		}
		s.applyOpponentMove(w, r, gameID)
	case "opponent-move-manual":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
			return
		}
		s.applyManualOpponentMove(w, r, gameID)
	default:
		writeError(w, http.StatusNotFound, errors.New("unknown game resource"))
	}
}

func (s *Server) getGameState(w http.ResponseWriter, r *http.Request, id domain.GameID) {
	game, err := s.games.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	board, err := s.games.GetChessboard(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	history, err := s.games.GenerateHistory(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, gameStateDTO{
		Game:  newGameDTO(game),
		Board: newBoardDTO(board),
		Moves: newMovesDTO(history),
	})
}

func (s *Server) deleteGame(w http.ResponseWriter, r *http.Request, id domain.GameID) {
	if err := s.games.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) applyUserMove(w http.ResponseWriter, r *http.Request, id domain.GameID) {
	var req userMoveRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Steps) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("move steps required"))
		return
	}
	trajectory := make([]domain.Position, 0, len(req.Steps))
	for _, step := range req.Steps {
		trajectory = append(trajectory, domain.Position{Row: domain.Coordinate(step.Row), Col: domain.Coordinate(step.Col)})
	}
	end := trajectory[len(trajectory)-1]
	preBoard, err := s.games.GetChessboard(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	cmd := sdkusecase.ProcessMoveCommand{
		GameID:        id,
		StartPosition: domain.Position{Row: domain.Coordinate(req.Start.Row), Col: domain.Coordinate(req.Start.Col)},
		Trajectory:    trajectory,
		EndPosition:   &end,
	}
	move, err := s.games.ProcessMove(r.Context(), cmd)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	board, err := s.games.GetChessboard(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	userAnimation := s.buildAnimation(preBoard, move)
	resp := moveOutcomeDTO{
		UserMove: newMoveDTOPtr(move),
		Board:    newBoardDTO(board),
		UserAnim: userAnimation,
	}
	if req.RequestOpponent {
		difficulty := parseDifficulty(req.OpponentDifficulty)
		opp, err := s.moves.GetOpponentMove(r.Context(), sdkusecase.GetOpponentMoveCommand{GameID: id, Difficulty: difficulty})
		if err != nil {
			writeJSON(w, http.StatusOK, resp)
			return
		}
		opponentAnimation := s.buildAnimation(board, opp)
		updatedBoard, _ := s.games.GetChessboard(r.Context(), id)
		resp.OpponentMove = newMoveDTOPtr(opp)
		resp.Board = newBoardDTO(updatedBoard)
		resp.OpponentAnim = opponentAnimation
	}
	history, err := s.games.GenerateHistory(r.Context(), id)
	if err == nil {
		resp.History = newMovesDTO(history)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) applyOpponentMove(w http.ResponseWriter, r *http.Request, id domain.GameID) {
	var req opponentMoveRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	difficulty := parseDifficulty(req.Difficulty)
	preBoard, err := s.games.GetChessboard(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	move, err := s.moves.GetOpponentMove(r.Context(), sdkusecase.GetOpponentMoveCommand{GameID: id, Difficulty: difficulty})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	board, err := s.games.GetChessboard(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	opponentAnimation := s.buildAnimation(preBoard, move)
	history, err := s.games.GenerateHistory(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, moveOutcomeDTO{
		OpponentMove: newMoveDTOPtr(move),
		Board:        newBoardDTO(board),
		History:      newMovesDTO(history),
		OpponentAnim: opponentAnimation,
	})
}

func (s *Server) applyManualOpponentMove(w http.ResponseWriter, r *http.Request, id domain.GameID) {
	var req userMoveRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Steps) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("move steps required"))
		return
	}
	trajectory := make([]domain.Position, 0, len(req.Steps))
	for _, step := range req.Steps {
		trajectory = append(trajectory, domain.Position{Row: domain.Coordinate(step.Row), Col: domain.Coordinate(step.Col)})
	}
	end := trajectory[len(trajectory)-1]
	preBoard, err := s.games.GetChessboard(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	cmd := sdkusecase.AddOpponentMoveCommand{
		GameID:        id,
		StartPosition: domain.Position{Row: domain.Coordinate(req.Start.Row), Col: domain.Coordinate(req.Start.Col)},
		Trajectory:    trajectory,
		EndPosition:   &end,
	}
	move, err := s.moves.AddOpponentMove(r.Context(), cmd)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	board, err := s.games.GetChessboard(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	opponentAnimation := s.buildAnimation(preBoard, move)
	history, err := s.games.GenerateHistory(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, moveOutcomeDTO{
		OpponentMove: newMoveDTOPtr(move),
		Board:        newBoardDTO(board),
		History:      newMovesDTO(history),
		OpponentAnim: opponentAnimation,
	})
}

func parseDifficulty(value string) domain.DifficultyLevel {
	switch strings.ToLower(value) {
	case string(domain.DifficultyEasy):
		return domain.DifficultyEasy
	case string(domain.DifficultyHard):
		return domain.DifficultyHard
	case string(domain.DifficultyMedium):
		fallthrough
	default:
		return domain.DifficultyMedium
	}
}

type createUserRequest struct {
	Name string `json:"name"`
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Rating     *int    `json:"rating"`
	WinStreak  *int    `json:"winStreak"`
	LastGameID *string `json:"lastGameId"`
}

type createGameRequest struct {
	UserID        string `json:"userId"`
	PlayerColor   string `json:"playerColor"`
	IsPlayerFirst bool   `json:"isPlayerFirst"`
}

type userMoveRequest struct {
	Start              positionDTO   `json:"start"`
	Steps              []positionDTO `json:"steps"`
	RequestOpponent    bool          `json:"requestOpponent"`
	OpponentDifficulty string        `json:"opponentDifficulty"`
}

type opponentMoveRequest struct {
	Difficulty string `json:"difficulty"`
}

type positionDTO struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type userDTO struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Rating     int     `json:"rating"`
	WinStreak  int     `json:"winStreak"`
	LastGameID *string `json:"lastGameId"`
}

type authResponse struct {
	User userDTO `json:"user"`
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	captureBody bool
	body        *limitedBuffer
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(p []byte) (int, error) {
	if r.captureBody && r.body != nil {
		_, _ = r.body.Write(p)
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

type httpLogConfig struct {
	level        string
	logBodies    bool
	maxBodyBytes int
}

func loadHTTPLogConfigFromEnv() httpLogConfig {
	level := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if level == "" {
		level = "info"
	}
	logBodies := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_HTTP_BODY"))) == "1" ||
		strings.ToLower(strings.TrimSpace(os.Getenv("LOG_HTTP_BODY"))) == "true" ||
		level == "debug"

	maxBodyBytes := 64 * 1024
	if raw := strings.TrimSpace(os.Getenv("LOG_MAX_BODY_BYTES")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			maxBodyBytes = parsed
		}
	}
	return httpLogConfig{level: level, logBodies: logBodies, maxBodyBytes: maxBodyBytes}
}

type teeReadCloser struct {
	rc  io.ReadCloser
	dst io.Writer
}

func (t *teeReadCloser) Read(p []byte) (int, error) {
	n, err := t.rc.Read(p)
	if n > 0 && t.dst != nil {
		_, _ = t.dst.Write(p[:n])
	}
	return n, err
}

func (t *teeReadCloser) Close() error {
	return t.rc.Close()
}

type limitedBuffer struct {
	buf      bytes.Buffer
	maxBytes int
}

func newLimitedBuffer(maxBytes int) *limitedBuffer {
	return &limitedBuffer{maxBytes: maxBytes}
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.maxBytes <= 0 {
		return len(p), nil
	}
	remaining := l.maxBytes - l.buf.Len()
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = l.buf.Write(p[:remaining])
		return len(p), nil
	}
	_, _ = l.buf.Write(p)
	return len(p), nil
}

func (l *limitedBuffer) String() string {
	return l.buf.String()
}

func compactJSON(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(text)); err == nil {
		return buf.String()
	}
	return text
}

type gameDTO struct {
	ID            string  `json:"id"`
	Status        string  `json:"status"`
	UserID        string  `json:"userId"`
	PlayerColor   string  `json:"playerColor"`
	IsPlayerFirst bool    `json:"isPlayerFirst"`
	StartedAt     string  `json:"startedAt"`
	FinishedAt    *string `json:"finishedAt"`
}

type gamesListDTO struct {
	Games []gameDTO `json:"games"`
}

type pieceDTO struct {
	ID    string `json:"id"`
	Row   int    `json:"row"`
	Col   int    `json:"col"`
	Color string `json:"color"`
	Kind  string `json:"kind"`
}

type boardDTO struct {
	Size   int        `json:"size"`
	Pieces []pieceDTO `json:"pieces"`
}

type moveDTO struct {
	ID        string        `json:"id"`
	Number    int           `json:"number"`
	PieceID   string        `json:"pieceId"`
	Start     positionDTO   `json:"start"`
	Path      []positionDTO `json:"path"`
	CreatedAt string        `json:"createdAt"`
}

type gameStateDTO struct {
	Game  gameDTO   `json:"game"`
	Board boardDTO  `json:"board"`
	Moves []moveDTO `json:"moves"`
}

type moveOutcomeDTO struct {
	UserMove     *moveDTO      `json:"userMove,omitempty"`
	OpponentMove *moveDTO      `json:"opponentMove,omitempty"`
	Board        boardDTO      `json:"board"`
	History      []moveDTO     `json:"history,omitempty"`
	UserAnim     *animationDTO `json:"userAnimation,omitempty"`
	OpponentAnim *animationDTO `json:"opponentAnimation,omitempty"`
}

type animationStepDTO struct {
	PieceID string      `json:"pieceId"`
	From    positionDTO `json:"from"`
	To      positionDTO `json:"to"`
}

type animationDTO struct {
	Steps []animationStepDTO `json:"steps"`
}

func newUserDTO(user domain.User) userDTO {
	var lastGame *string
	if user.LastGameID != nil {
		value := string(*user.LastGameID)
		lastGame = &value
	}
	return userDTO{
		ID:         string(user.ID),
		Name:       string(user.Name),
		Rating:     int(user.Rating),
		WinStreak:  int(user.WinStreak),
		LastGameID: lastGame,
	}
}

func newGameDTO(game domain.Game) gameDTO {
	var finished *string
	if game.End != nil {
		value := game.End.Time.Format(time.RFC3339)
		finished = &value
	}
	return gameDTO{
		ID:            string(game.ID),
		Status:        string(game.Status),
		UserID:        string(game.UserID),
		PlayerColor:   string(game.PlayerColor),
		IsPlayerFirst: game.IsPlayerFirst,
		StartedAt:     game.Start.Time.Format(time.RFC3339),
		FinishedAt:    finished,
	}
}

func newBoardDTO(board domain.BoardState) boardDTO {
	pieces := make([]pieceDTO, 0, len(board.Pieces))
	for _, piece := range board.Pieces {
		pieces = append(pieces, pieceDTO{
			ID:    string(piece.ID),
			Row:   int(piece.Position.Row),
			Col:   int(piece.Position.Col),
			Color: string(piece.Color),
			Kind:  string(piece.Kind),
		})
	}
	return boardDTO{
		Size:   int(board.Size),
		Pieces: pieces,
	}
}

func newMoveDTO(move domain.Move) moveDTO {
	path := make([]positionDTO, 0, len(move.Trajectory))
	for _, pos := range move.Trajectory {
		path = append(path, positionDTO{Row: int(pos.Row), Col: int(pos.Col)})
	}
	return moveDTO{
		ID:        string(move.ID),
		Number:    int(move.Number),
		PieceID:   string(move.PieceID),
		Start:     positionDTO{Row: int(move.StartPosition.Row), Col: int(move.StartPosition.Col)},
		Path:      path,
		CreatedAt: move.CreatedAt.Time.Format(time.RFC3339),
	}
}

func newMoveDTOPtr(move domain.Move) *moveDTO {
	dto := newMoveDTO(move)
	return &dto
}

func newMovesDTO(moves []domain.Move) []moveDTO {
	result := make([]moveDTO, 0, len(moves))
	for _, move := range moves {
		result = append(result, newMoveDTO(move))
	}
	return result
}

func (s *Server) buildAnimation(board domain.BoardState, move domain.Move) *animationDTO {
	if s.anim == nil {
		return nil
	}
	animation, err := s.anim.Build(context.Background(), sdkusecase.MoveAnimationCommand{
		Board: board,
		Move:  move,
	})
	if err != nil || len(animation.Steps) == 0 {
		return nil
	}
	steps := make([]animationStepDTO, 0, len(animation.Steps))
	for _, step := range animation.Steps {
		steps = append(steps, animationStepDTO{
			PieceID: string(step.PieceID),
			From: positionDTO{
				Row: int(step.From.Row),
				Col: int(step.From.Col),
			},
			To: positionDTO{
				Row: int(step.To.Row),
				Col: int(step.To.Col),
			},
		})
	}
	return &animationDTO{Steps: steps}
}

func newGamesDTO(games []domain.Game) []gameDTO {
	items := make([]gameDTO, 0, len(games))
	for _, g := range games {
		items = append(items, newGameDTO(g))
	}
	return items
}

func decodeJSON(ctx context.Context, body io.ReadCloser, dst interface{}) error {
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
