package api

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ppo/sdk/domain"
	sdkusecase "ppo/sdk/usecase"
)

var (
	ErrAuthLocked            = errors.New("auth temporarily locked")
	ErrAuthChallengeNotFound = errors.New("auth challenge not found")
	ErrAuthChallengeExpired  = errors.New("auth challenge expired")
	ErrAuthInvalidCode       = errors.New("invalid verification code")
	ErrAuthUnauthorized      = errors.New("unauthorized")
	ErrAuthInvalidRequest    = errors.New("invalid request")
)

type codeSender interface {
	SendCode(ctx context.Context, login domain.Login, purpose string, code string) error
}

type loginStartResult struct {
	User        domain.User
	ChallengeID string
}

type challenge struct {
	ID        string
	Login     domain.Login
	User      domain.User
	Purpose   string
	Code      string
	ExpiresAt time.Time

	Attempts int

	OldPassword string
	NewPassword string
}

type session struct {
	Token     string
	Login     domain.Login
	UserID    domain.UserID
	ExpiresAt time.Time
}

type recoveryCode struct {
	Code      string
	ExpiresAt time.Time
}

type authFlow struct {
	mu sync.Mutex

	auth   sdkusecase.AuthUseCase
	sender codeSender

	challenges map[string]challenge
	sessions   map[string]session
	recovery   map[domain.Login]recoveryCode

	failed      map[domain.Login]int
	lockedUntil map[domain.Login]time.Time

	maxAttempts int
	lockFor     time.Duration
	codeTTL     time.Duration
	sessionTTL  time.Duration
}

func newAuthFlowFromEnv(auth sdkusecase.AuthUseCase, outbox *memoryOutbox) *authFlow {
	maxAttempts := envInt("AUTH_MAX_ATTEMPTS", 3)
	lockFor := envDuration("AUTH_LOCK_DURATION", 30*time.Second)
	codeTTL := envDuration("AUTH_CODE_TTL", 2*time.Minute)
	sessionTTL := envDuration("AUTH_SESSION_TTL", 15*time.Minute)

	var sender codeSender = &logSender{}
	// In test mode we still log, but also persist codes to an outbox so tests can fetch them.
	sender = &outboxSender{outbox: outbox, next: sender}

	return &authFlow{
		auth:        auth,
		sender:      sender,
		challenges:  make(map[string]challenge),
		sessions:    make(map[string]session),
		recovery:    make(map[domain.Login]recoveryCode),
		failed:      make(map[domain.Login]int),
		lockedUntil: make(map[domain.Login]time.Time),
		maxAttempts: maxAttempts,
		lockFor:     lockFor,
		codeTTL:     codeTTL,
		sessionTTL:  sessionTTL,
	}
}

func (f *authFlow) StartLogin(ctx context.Context, login domain.Login, password string) (loginStartResult, error) {
	login = domain.Login(strings.TrimSpace(strings.ToLower(string(login))))
	password = strings.TrimSpace(password)
	if login == "" || password == "" {
		return loginStartResult{}, ErrAuthInvalidRequest
	}
	if f.auth == nil {
		return loginStartResult{}, ErrAuthUnauthorized
	}

	f.mu.Lock()
	if f.isLocked(login) {
		f.mu.Unlock()
		return loginStartResult{}, ErrAuthLocked
	}
	f.mu.Unlock()

	user, err := f.auth.Login(ctx, sdkusecase.LoginCommand{Login: login, Password: password})
	if err != nil {
		return loginStartResult{}, ErrAuthUnauthorized
	}

	chID := randomID("ch")
	code := randomCode()
	ch := challenge{
		ID:        chID,
		Login:     login,
		User:      user,
		Purpose:   "login",
		Code:      code,
		ExpiresAt: time.Now().Add(f.codeTTL),
	}

	f.mu.Lock()
	f.challenges[chID] = ch
	f.mu.Unlock()

	_ = f.sender.SendCode(ctx, login, ch.Purpose, code)
	return loginStartResult{User: user, ChallengeID: chID}, nil
}

func (f *authFlow) VerifyLogin(ctx context.Context, challengeID string, code string) (session, domain.User, error) {
	challengeID = strings.TrimSpace(challengeID)
	code = strings.TrimSpace(code)
	if challengeID == "" || code == "" {
		return session{}, domain.User{}, ErrAuthInvalidRequest
	}

	f.mu.Lock()
	ch, ok := f.challenges[challengeID]
	if !ok {
		f.mu.Unlock()
		return session{}, domain.User{}, ErrAuthChallengeNotFound
	}
	if time.Now().After(ch.ExpiresAt) {
		delete(f.challenges, challengeID)
		f.mu.Unlock()
		return session{}, domain.User{}, ErrAuthChallengeExpired
	}
	if f.isLocked(ch.Login) {
		f.mu.Unlock()
		return session{}, domain.User{}, ErrAuthLocked
	}
	if ch.Code != code {
		ch.Attempts++
		if ch.Attempts >= f.maxAttempts {
			f.lockedUntil[ch.Login] = time.Now().Add(f.lockFor)
			delete(f.challenges, challengeID)
			f.mu.Unlock()
			return session{}, domain.User{}, ErrAuthLocked
		}
		f.challenges[challengeID] = ch
		f.mu.Unlock()
		return session{}, domain.User{}, ErrAuthInvalidCode
	}

	// success
	delete(f.challenges, challengeID)
	f.failed[ch.Login] = 0
	token := randomID("token")
	sess := session{
		Token:     token,
		Login:     ch.Login,
		UserID:    ch.User.ID,
		ExpiresAt: time.Now().Add(f.sessionTTL),
	}
	f.sessions[token] = sess
	f.mu.Unlock()
	return sess, ch.User, nil
}

func (f *authFlow) StartPasswordChange(ctx context.Context, token string, oldPassword string, newPassword string) (string, error) {
	sess, err := f.requireSession(token)
	if err != nil {
		return "", err
	}
	oldPassword = strings.TrimSpace(oldPassword)
	newPassword = strings.TrimSpace(newPassword)
	if oldPassword == "" || newPassword == "" {
		return "", ErrAuthInvalidRequest
	}
	if f.auth == nil {
		return "", ErrAuthUnauthorized
	}

	// verify old password at request time
	if _, err := f.auth.Login(ctx, sdkusecase.LoginCommand{Login: sess.Login, Password: oldPassword}); err != nil {
		return "", ErrAuthUnauthorized
	}

	chID := randomID("pwd")
	code := randomCode()
	ch := challenge{
		ID:          chID,
		Login:       sess.Login,
		Purpose:     "password_change",
		Code:        code,
		ExpiresAt:   time.Now().Add(f.codeTTL),
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}

	f.mu.Lock()
	if f.isLocked(sess.Login) {
		f.mu.Unlock()
		return "", ErrAuthLocked
	}
	f.challenges[chID] = ch
	f.mu.Unlock()

	_ = f.sender.SendCode(ctx, sess.Login, ch.Purpose, code)
	return chID, nil
}

func (f *authFlow) ConfirmPasswordChange(ctx context.Context, challengeID string, code string) error {
	challengeID = strings.TrimSpace(challengeID)
	code = strings.TrimSpace(code)
	if challengeID == "" || code == "" {
		return ErrAuthInvalidRequest
	}
	if f.auth == nil {
		return ErrAuthUnauthorized
	}

	f.mu.Lock()
	ch, ok := f.challenges[challengeID]
	if !ok {
		f.mu.Unlock()
		return ErrAuthChallengeNotFound
	}
	if ch.Purpose != "password_change" {
		f.mu.Unlock()
		return ErrAuthInvalidRequest
	}
	if time.Now().After(ch.ExpiresAt) {
		delete(f.challenges, challengeID)
		f.mu.Unlock()
		return ErrAuthChallengeExpired
	}
	if f.isLocked(ch.Login) {
		f.mu.Unlock()
		return ErrAuthLocked
	}
	if ch.Code != code {
		ch.Attempts++
		if ch.Attempts >= f.maxAttempts {
			f.lockedUntil[ch.Login] = time.Now().Add(f.lockFor)
			delete(f.challenges, challengeID)
			f.mu.Unlock()
			return ErrAuthLocked
		}
		f.challenges[challengeID] = ch
		f.mu.Unlock()
		return ErrAuthInvalidCode
	}
	delete(f.challenges, challengeID)
	f.failed[ch.Login] = 0
	f.mu.Unlock()

	// We already validated old password when starting the change. Here we only apply the new password.
	return f.auth.ResetPassword(ctx, sdkusecase.ResetPasswordCommand{Login: ch.Login, NewPassword: ch.NewPassword})
}

func (f *authFlow) RequestRecovery(ctx context.Context, login domain.Login) error {
	login = domain.Login(strings.TrimSpace(strings.ToLower(string(login))))
	if login == "" {
		return ErrAuthInvalidRequest
	}
	code := randomCode()

	f.mu.Lock()
	f.recovery[login] = recoveryCode{
		Code:      code,
		ExpiresAt: time.Now().Add(f.codeTTL),
	}
	f.mu.Unlock()

	_ = f.sender.SendCode(ctx, login, "recovery", code)
	return nil
}

func (f *authFlow) ConfirmRecovery(ctx context.Context, login domain.Login, code string, newPassword string) error {
	login = domain.Login(strings.TrimSpace(strings.ToLower(string(login))))
	code = strings.TrimSpace(code)
	newPassword = strings.TrimSpace(newPassword)
	if login == "" || code == "" || newPassword == "" {
		return ErrAuthInvalidRequest
	}
	if f.auth == nil {
		return ErrAuthUnauthorized
	}

	f.mu.Lock()
	rc, ok := f.recovery[login]
	if !ok {
		f.mu.Unlock()
		return ErrAuthInvalidCode
	}
	if time.Now().After(rc.ExpiresAt) {
		delete(f.recovery, login)
		f.mu.Unlock()
		return ErrAuthChallengeExpired
	}
	if rc.Code != code {
		err := f.recordFailedLocked(login)
		f.mu.Unlock()
		if err != nil {
			return err
		}
		return ErrAuthInvalidCode
	}
	delete(f.recovery, login)
	f.failed[login] = 0
	f.lockedUntil[login] = time.Time{}
	f.mu.Unlock()

	return f.auth.ResetPassword(ctx, sdkusecase.ResetPasswordCommand{Login: login, NewPassword: newPassword})
}

func (f *authFlow) requireSession(token string) (session, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return session{}, ErrAuthUnauthorized
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	sess, ok := f.sessions[token]
	if !ok {
		return session{}, ErrAuthUnauthorized
	}
	if time.Now().After(sess.ExpiresAt) {
		delete(f.sessions, token)
		return session{}, ErrAuthUnauthorized
	}
	if f.isLocked(sess.Login) {
		return session{}, ErrAuthLocked
	}
	return sess, nil
}

func (f *authFlow) isLocked(login domain.Login) bool {
	until, ok := f.lockedUntil[login]
	if !ok || until.IsZero() {
		return false
	}
	if time.Now().After(until) {
		f.lockedUntil[login] = time.Time{}
		f.failed[login] = 0
		return false
	}
	return true
}

func (f *authFlow) recordFailedLocked(login domain.Login) error {
	f.failed[login]++
	if f.failed[login] >= f.maxAttempts {
		f.lockedUntil[login] = time.Now().Add(f.lockFor)
		return ErrAuthLocked
	}
	return nil
}

func authHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrAuthInvalidRequest):
		return http.StatusBadRequest
	case errors.Is(err, ErrAuthLocked):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrAuthChallengeNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrAuthChallengeExpired):
		return http.StatusGone
	case errors.Is(err, ErrAuthInvalidCode), errors.Is(err, ErrAuthUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func randomID(prefix string) string {
	buf := make([]byte, 10)
	_, _ = rand.Read(buf)
	return prefix + "_" + strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), "=")
}

func randomCode() string {
	// 6 digits
	var b [4]byte
	_, _ = rand.Read(b[:])
	n := int(b[0])<<8 | int(b[1])
	n = n % 1000000
	code := strconv.Itoa(n)
	return strings.Repeat("0", 6-len(code)) + code
}

func envBool(key string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func envInt(key string, def int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return parsed
}

func envDuration(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return def
	}
	return parsed
}
