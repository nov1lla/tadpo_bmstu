package localjson

import (
	"errors"
	"testing"

	"ppo/sdk/port/repo"
)

func TestUserCredentialsRepositorySave_OK(t *testing.T) {
	fixture := newCredentialsRepoFixture(t)
	creds := MotherCredentials()

	err := fixture.repo.Save(fixture.ctx, creds)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	loaded, err := fixture.repo.GetByLogin(fixture.ctx, creds.Login)
	if err != nil {
		t.Fatalf("get by login: %v", err)
	}
	if loaded.Login != creds.Login || loaded.UserID != creds.UserID {
		t.Fatalf("unexpected creds: %+v", loaded)
	}
}

func TestUserCredentialsRepositorySave_RejectsDuplicate(t *testing.T) {
	fixture := newCredentialsRepoFixture(t)
	creds := NewCredentialsBuilder().WithLogin("login-1").Build()
	if err := fixture.repo.Save(fixture.ctx, creds); err != nil {
		t.Fatalf("save: %v", err)
	}

	err := fixture.repo.Save(fixture.ctx, creds)

	if !errors.Is(err, repo.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUserCredentialsRepositoryGetByLogin_NotFound(t *testing.T) {
	fixture := newCredentialsRepoFixture(t)

	_, err := fixture.repo.GetByLogin(fixture.ctx, "missing")

	if !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
