package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"ppo/sdk/domain"
)

type outboxMessage struct {
	Login     domain.Login
	Purpose   string
	Code      string
	CreatedAt time.Time
}

type memoryOutbox struct {
	mu       sync.Mutex
	messages map[string]outboxMessage
}

func newMemoryOutbox() *memoryOutbox {
	return &memoryOutbox{messages: make(map[string]outboxMessage)}
}

func (o *memoryOutbox) Put(msg outboxMessage) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.messages[o.key(msg.Login, msg.Purpose)] = msg
}

func (o *memoryOutbox) Latest(login domain.Login, purpose string) (outboxMessage, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	msg, ok := o.messages[o.key(login, purpose)]
	return msg, ok
}

func (o *memoryOutbox) key(login domain.Login, purpose string) string {
	return strings.ToLower(string(login)) + "::" + strings.ToLower(strings.TrimSpace(purpose))
}

type logSender struct{}

func (s *logSender) SendCode(_ context.Context, login domain.Login, purpose string, code string) error {
	log.Printf("2fa code purpose=%s login=%s code=%s", purpose, login, code)
	return nil
}

type outboxSender struct {
	outbox *memoryOutbox
	next   codeSender
}

func (s *outboxSender) SendCode(ctx context.Context, login domain.Login, purpose string, code string) error {
	if s.outbox != nil {
		s.outbox.Put(outboxMessage{
			Login:     login,
			Purpose:   purpose,
			Code:      code,
			CreatedAt: time.Now(),
		})
	}
	if s.next != nil {
		return s.next.SendCode(ctx, login, purpose, code)
	}
	return nil
}

type outboxResponse struct {
	Login     string `json:"login"`
	Purpose   string `json:"purpose"`
	Code      string `json:"code"`
	CreatedAt string `json:"createdAt"`
}

func (s *Server) handleTestOutbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		return
	}

	expected := strings.TrimSpace(os.Getenv("WEBAPP_TEST_TOKEN"))
	if expected == "" {
		expected = "test"
	}
	got := strings.TrimSpace(r.Header.Get("X-Test-Token"))
	if got == "" || got != expected {
		writeError(w, http.StatusUnauthorized, ErrAuthUnauthorized)
		return
	}

	login := domain.Login(strings.TrimSpace(r.URL.Query().Get("login")))
	purpose := strings.TrimSpace(r.URL.Query().Get("purpose"))
	if login == "" || purpose == "" {
		writeError(w, http.StatusBadRequest, ErrAuthInvalidRequest)
		return
	}

	msg, ok := s.authFlowOutboxLatest(login, purpose)
	if !ok {
		writeError(w, http.StatusNotFound, errors.New("message not found"))
		return
	}

	writeJSON(w, http.StatusOK, outboxResponse{
		Login:     string(msg.Login),
		Purpose:   msg.Purpose,
		Code:      msg.Code,
		CreatedAt: msg.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (s *Server) authFlowOutboxLatest(login domain.Login, purpose string) (outboxMessage, bool) {
	// authFlow is configured to always use outboxSender; we retrieve the outbox via type assertion.
	f := s.authFlow
	if f == nil {
		return outboxMessage{}, false
	}
	sender := f.sender
	obs, ok := sender.(*outboxSender)
	if !ok || obs.outbox == nil {
		return outboxMessage{}, false
	}
	return obs.outbox.Latest(login, purpose)
}
