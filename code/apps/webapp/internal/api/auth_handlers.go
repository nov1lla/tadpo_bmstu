package api

import (
	"errors"
	"net/http"
	"strings"

	"ppo/sdk/domain"
)

type authChallengeResponse struct {
	User        userDTO `json:"user"`
	ChallengeID string  `json:"challengeId"`
	TwoFactor   bool    `json:"twoFactor"`
}

type verifyChallengeRequest struct {
	ChallengeID string `json:"challengeId"`
	Code        string `json:"code"`
}

type verifyChallengeResponse struct {
	User  userDTO `json:"user"`
	Token string  `json:"token"`
}

type passwordChangeRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type passwordChangeStartResponse struct {
	ChallengeID string `json:"challengeId"`
}

type recoveryRequest struct {
	Login string `json:"login"`
}

type recoveryConfirmRequest struct {
	Login       string `json:"login"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

func (s *Server) handleLoginVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		return
	}
	var req verifyChallengeRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	sess, user, err := s.authFlow.VerifyLogin(r.Context(), req.ChallengeID, req.Code)
	if err != nil {
		writeError(w, authHTTPStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, verifyChallengeResponse{
		User:  newUserDTO(user),
		Token: sess.Token,
	})
}

func (s *Server) handlePasswordChangeRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	var req passwordChangeRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	chID, err := s.authFlow.StartPasswordChange(r.Context(), token, req.OldPassword, req.NewPassword)
	if err != nil {
		writeError(w, authHTTPStatus(err), err)
		return
	}
	writeJSON(w, http.StatusAccepted, passwordChangeStartResponse{ChallengeID: chID})
}

func (s *Server) handlePasswordChangeConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		return
	}
	var req verifyChallengeRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.authFlow.ConfirmPasswordChange(r.Context(), req.ChallengeID, req.Code); err != nil {
		writeError(w, authHTTPStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRecoveryRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		return
	}
	var req recoveryRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Always respond with 200 to avoid login enumeration.
	_ = s.authFlow.RequestRecovery(r.Context(), domain.Login(req.Login))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRecoveryConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errors.New("unsupported method"))
		return
	}
	var req recoveryConfirmRequest
	if err := decodeJSON(r.Context(), r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.authFlow.ConfirmRecovery(r.Context(), domain.Login(req.Login), req.Code, req.NewPassword); err != nil {
		writeError(w, authHTTPStatus(err), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
