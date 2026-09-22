package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"wallet-app/internal/core/domain"
	"wallet-app/internal/features/auth/service"

	"github.com/google/uuid"
)

type repositoryStub struct {
	t      *testing.T
	create func(context.Context, domain.User) error
	get    func(context.Context, string) (domain.User, error)
}

func (r *repositoryStub) Create(ctx context.Context, user domain.User) error {
	r.t.Helper()
	if r.create == nil {
		r.t.Fatal("unexpected Create call")
	}
	return r.create(ctx, user)
}

func (r *repositoryStub) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	r.t.Helper()
	if r.get == nil {
		r.t.Fatal("unexpected GetByUsername call")
	}
	return r.get(ctx, username)
}

type hasherStub struct {
	maxByteLength int
	t             *testing.T
	hash          func(string) (string, error)
	compare       func(string, string) error
}

func (h *hasherStub) MaxByteLength() int { return h.maxByteLength }

func (h *hasherStub) Hash(password string) (string, error) {
	h.t.Helper()
	if h.hash == nil {
		h.t.Fatal("unexpected Hash call")
	}
	return h.hash(password)
}

func (h *hasherStub) Compare(password, hash string) error {
	h.t.Helper()
	if h.compare == nil {
		h.t.Fatal("unexpected Compare call")
	}
	return h.compare(password, hash)
}

func TestRegister(t *testing.T) {
	validID := uuid.MustParse("e373039e-7060-4525-90bc-d5c7d7525a80")
	hashErr := errors.New("hash failed")
	repoErr := errors.New("database unavailable")
	for _, tt := range []struct {
		name                                      string
		username, email, password                 string
		nilID, emptyHash, canceled, emptyPassword bool
		hashErr, repoErr, wantErr                 error
		wantRepoCalls                             int
		maxByteLength                             int
	}{
		{name: "success", wantRepoCalls: 1},
		{name: "password spaces preserved", password: " secret ", wantRepoCalls: 1},
		{name: "72 ASCII bytes", password: strings.Repeat("a", 72), wantRepoCalls: 1},
		{name: "73 ASCII bytes", password: strings.Repeat("a", 73), wantErr: domain.ErrInvalidPassword},
		{name: "72 Unicode bytes", password: strings.Repeat("я", 36), wantRepoCalls: 1},
		{name: "73 Unicode bytes", password: strings.Repeat("я", 36) + "a", wantErr: domain.ErrInvalidPassword},
		{name: "spaces count toward limit", password: " " + strings.Repeat("a", 72), wantErr: domain.ErrInvalidPassword},
		{name: "custom limit accepted", maxByteLength: 8, password: "12345678", wantRepoCalls: 1},
		{name: "custom limit exceeded", maxByteLength: 8, password: "123456789", wantErr: domain.ErrInvalidPassword},
		{name: "empty password", emptyPassword: true, wantErr: domain.ErrInvalidPassword},
		{name: "whitespace password", password: " \t\n\u2003", wantErr: domain.ErrInvalidPassword},
		{name: "blank username", username: " \t", wantErr: domain.ErrInvalidUsername},
		{name: "invalid email", email: "invalid", wantErr: domain.ErrInvalidEmailAddress},
		{name: "nil ID", nilID: true, wantErr: domain.ErrInvalidUserID},
		{name: "empty hash", emptyHash: true, wantErr: domain.ErrInvalidPasswordHash},
		{name: "hash error", hashErr: hashErr, wantErr: hashErr},
		{name: "duplicate username", repoErr: domain.ErrUsernameAlreadyExists, wantErr: domain.ErrUsernameAlreadyExists, wantRepoCalls: 1},
		{name: "duplicate email", repoErr: domain.ErrEmailAlreadyExists, wantErr: domain.ErrEmailAlreadyExists, wantRepoCalls: 1},
		{name: "repository error", repoErr: repoErr, wantErr: repoErr, wantRepoCalls: 1},
		{name: "canceled context", canceled: true, wantErr: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			username, email, password := tt.username, tt.email, tt.password
			if username == "" {
				username = " IvAn "
			}
			if email == "" {
				email = "Ivan@example.com"
			}
			if password == "" {
				password = "secret"
			}
			if tt.emptyPassword {
				password = ""
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			generatedID := validID
			if tt.nilID {
				generatedID = uuid.Nil
			}
			hashValue := "stored-password-hash"
			if tt.emptyHash {
				hashValue = ""
			}
			hashCalls, generatorCalls, repoCalls := 0, 0, 0
			repo := &repositoryStub{t: t, create: func(gotCtx context.Context, user domain.User) error {
				repoCalls++
				if gotCtx != ctx {
					t.Error("repository received a different context")
				}
				if user.ID() != validID || user.Username() != "ivan" || user.Email() != "Ivan@example.com" || user.PasswordHash() != hashValue {
					t.Error("repository received incorrect user fields")
				}
				return tt.repoErr
			}}
			limit := tt.maxByteLength
			if limit == 0 {
				limit = 72
			}
			hasher := &hasherStub{t: t, maxByteLength: limit, hash: func(got string) (string, error) {
				hashCalls++
				if got != password {
					t.Error("password was modified before hashing")
				}
				return hashValue, tt.hashErr
			}}
			svc := service.New(repo, func() uuid.UUID { generatorCalls++; return generatedID }, hasher, nil)
			got, err := svc.Register(ctx, username, email, password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v; want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got != (domain.User{}) {
					t.Error("failure returned a nonzero user")
				}
			} else if got.ID() != validID || got.Username() != "ivan" || got.Email() != "Ivan@example.com" || got.PasswordHash() != hashValue {
				t.Error("registration returned incorrect user fields")
			}
			if repoCalls != tt.wantRepoCalls {
				t.Errorf("Create calls = %d; want %d", repoCalls, tt.wantRepoCalls)
			}
			if tt.canceled || errors.Is(tt.wantErr, domain.ErrInvalidPassword) {
				if hashCalls != 0 || generatorCalls != 0 {
					t.Error("rejected request invoked hasher or ID generator")
				}
			}
			if tt.hashErr != nil && generatorCalls != 0 {
				t.Error("ID generated after hash failure")
			}
			if tt.wantRepoCalls == 1 && (hashCalls != 1 || generatorCalls != 1) {
				t.Error("expected exactly one hash and ID generation")
			}
		})
	}
}

type jwtStub struct {
	t        *testing.T
	generate func(string) (string, error)
}

func (g *jwtStub) Generate(subject string) (string, error) {
	g.t.Helper()
	return g.generate(subject)
}

func (g *jwtStub) Validate(string) (string, error) {
	g.t.Helper()
	g.t.Fatal("unexpected Validate call during login")
	return "", nil
}

func TestLogin(t *testing.T) {
	user, err := domain.NewUser(uuid.MustParse("e373039e-7060-4525-90bc-d5c7d7525a80"), "ivan", "Ivan@example.com", "stored-hash")
	if err != nil {
		t.Fatal(err)
	}
	repoErr := errors.New("database unavailable")
	tokenErr := errors.New("token generation failed")
	compareErr := errors.New("hasher unavailable")
	malformedHashErr := errors.New("malformed stored hash")
	for _, tt := range []struct {
		name, password               string
		canceled                     bool
		tokenErr                     error
		repoErr, compareErr, wantErr error
		wantGet, wantCompare         int
	}{
		{name: "normalized username and unchanged password", password: " secret ", wantGet: 1, wantCompare: 1},
		{name: "token generation failure", password: "secret", tokenErr: tokenErr, wantErr: tokenErr, wantGet: 1, wantCompare: 1},
		{name: "missing user", password: "secret", repoErr: fmt.Errorf("lookup: %w", domain.ErrUserNotFound), wantErr: domain.ErrInvalidUserCredentials, wantGet: 1},
		{name: "wrong password", password: "wrong", compareErr: fmt.Errorf("compare: %w", domain.ErrPasswordMismatch), wantErr: domain.ErrInvalidUserCredentials, wantGet: 1, wantCompare: 1},
		{name: "repository failure", password: "secret", repoErr: repoErr, wantErr: repoErr, wantGet: 1},
		{name: "repository cancellation", password: "secret", repoErr: context.Canceled, wantErr: context.Canceled, wantGet: 1},
		{name: "hasher failure", password: "secret", compareErr: compareErr, wantErr: compareErr, wantGet: 1, wantCompare: 1},
		{name: "malformed stored hash", password: "secret", compareErr: malformedHashErr, wantErr: malformedHashErr, wantGet: 1, wantCompare: 1},
		{name: "empty password", wantErr: domain.ErrInvalidPassword},
		{name: "whitespace password", password: " \t\n", wantErr: domain.ErrInvalidPassword},
		{name: "73-byte password", password: strings.Repeat("я", 36) + "a", wantErr: domain.ErrInvalidPassword},
		{name: "72-byte password", password: strings.Repeat("я", 36), wantGet: 1, wantCompare: 1},
		{name: "canceled context", password: "secret", canceled: true, wantErr: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			getCalls, compareCalls := 0, 0
			repo := &repositoryStub{t: t, get: func(gotCtx context.Context, username string) (domain.User, error) {
				getCalls++
				if gotCtx != ctx || username != "ivan" {
					t.Error("incorrect context or username normalization")
				}
				if tt.repoErr != nil {
					return domain.User{}, tt.repoErr
				}
				return user, nil
			}}
			hasher := &hasherStub{t: t, maxByteLength: 72, compare: func(password, hash string) error {
				compareCalls++
				if password != tt.password || hash != user.PasswordHash() {
					t.Error("incorrect Compare arguments")
				}
				return tt.compareErr
			}}
			tokenCalls := 0
			generator := &jwtStub{t: t, generate: func(subject string) (string, error) {
				tokenCalls++
				if compareCalls != 1 || tt.compareErr != nil {
					t.Error("token generated before successful password verification")
				}
				if subject != user.ID().String() {
					t.Error("incorrect token subject")
				}
				if tt.tokenErr != nil {
					return "", tt.tokenErr
				}
				return "signed-token", nil
			}}
			svc := service.New(repo, func() uuid.UUID { t.Fatal("Login must not generate an ID"); return uuid.Nil }, hasher, generator)
			got, gotToken, err := svc.Login(ctx, " IvAn ", tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v; want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if gotToken != "" {
					t.Error("failed login returned a token")
				}
				if got != (domain.User{}) {
					t.Error("failed login returned a user")
				}
			} else if got != user {
				t.Error("Login returned a different user")
			}
			if tt.wantErr == nil && gotToken != "signed-token" {
				t.Errorf("token = %q; want signed-token", gotToken)
			}
			wantTokenCalls := 0
			if tt.wantErr == nil || tt.tokenErr != nil {
				wantTokenCalls = 1
			}
			if tokenCalls != wantTokenCalls {
				t.Errorf("Generate calls = %d; want %d", tokenCalls, wantTokenCalls)
			}
			if getCalls != tt.wantGet || compareCalls != tt.wantCompare {
				t.Errorf("GetByUsername/Compare calls = %d/%d; want %d/%d", getCalls, compareCalls, tt.wantGet, tt.wantCompare)
			}
		})
	}
}
