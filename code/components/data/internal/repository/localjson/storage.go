package localjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	usersFile       = "users.json"
	credentialsFile = "credentials.json"
	gamesFile       = "games.json"
	boardStatesFile = "board_states.json"
	movesFile       = "moves.json"
)

type Storage struct {
	files *fileStorage
}

type fileStorage struct {
	dir string
	mu  sync.Mutex
}

func NewStorage(dir string) (*Storage, error) {
	files, err := newFileStorage(dir)
	if err != nil {
		return nil, err
	}
	return &Storage{files: files}, nil
}

func newFileStorage(dir string) (*fileStorage, error) {
	if dir == "" {
		return nil, fmt.Errorf("storage directory is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}
	return &fileStorage{dir: dir}, nil
}

func (s *fileStorage) loadUsers() (map[string]storedUser, error) {
	var payload map[string]storedUser
	if err := s.load(usersFile, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = make(map[string]storedUser)
	}
	return payload, nil
}

func (s *fileStorage) persistUsers(data map[string]storedUser) error {
	return s.persist(usersFile, data)
}

func (s *fileStorage) loadCredentials() (map[string]storedUserCredentials, error) {
	var payload map[string]storedUserCredentials
	if err := s.load(credentialsFile, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = make(map[string]storedUserCredentials)
	}
	return payload, nil
}

func (s *fileStorage) persistCredentials(data map[string]storedUserCredentials) error {
	return s.persist(credentialsFile, data)
}

func (s *fileStorage) loadGames() (map[string]storedGame, error) {
	var payload map[string]storedGame
	if err := s.load(gamesFile, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = make(map[string]storedGame)
	}
	return payload, nil
}

func (s *fileStorage) persistGames(data map[string]storedGame) error {
	return s.persist(gamesFile, data)
}

func (s *fileStorage) loadBoardStates() (map[string]storedBoardState, error) {
	var payload map[string]storedBoardState
	if err := s.load(boardStatesFile, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = make(map[string]storedBoardState)
	}
	return payload, nil
}

func (s *fileStorage) persistBoardStates(data map[string]storedBoardState) error {
	return s.persist(boardStatesFile, data)
}

func (s *fileStorage) loadMoves() (map[string]storedMove, error) {
	var payload map[string]storedMove
	if err := s.load(movesFile, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = make(map[string]storedMove)
	}
	return payload, nil
}

func (s *fileStorage) persistMoves(data map[string]storedMove) error {
	return s.persist(movesFile, data)
}

func (s *fileStorage) load(filename string, out interface{}) error {
	path := filepath.Join(s.dir, filename)
	bytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read %s: %w", filename, err)
	}
	if len(bytes) == 0 {
		return nil
	}
	if err := json.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decode %s: %w", filename, err)
	}
	return nil
}

func (s *fileStorage) persist(filename string, data interface{}) error {
	path := filepath.Join(s.dir, filename)
	tempPath := path + ".tmp"
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filename, err)
	}
	if err := os.WriteFile(tempPath, bytes, 0o644); err != nil {
		return fmt.Errorf("write temp %s: %w", filename, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", filename, err)
	}
	return nil
}
