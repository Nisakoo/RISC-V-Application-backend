package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

const testPassword = "riscv-test-password"

type fakeUserRepository struct {
	users        map[string]domain.User
	registered   string
	passwordHash string
}

func (fake *fakeUserRepository) RegisterUser(_ context.Context, login, passwordHash string) (int64, error) {
	fake.registered = login
	fake.passwordHash = passwordHash
	return 7, nil
}

func (fake *fakeUserRepository) GetUserByLogin(_ context.Context, login string) (domain.User, error) {
	user, exists := fake.users[login]
	if !exists {
		return domain.User{}, repository.ErrUserNotFound
	}
	return user, nil
}

type fakeBlacklist struct {
	revoked map[string]time.Duration
	err     error
}

func (fake *fakeBlacklist) Add(_ context.Context, tokenID, _ string, ttl time.Duration) error {
	fake.revoked[tokenID] = ttl
	return nil
}

func (fake *fakeBlacklist) Contains(_ context.Context, tokenID string) (bool, error) {
	if fake.err != nil {
		return false, fake.err
	}
	_, exists := fake.revoked[tokenID]
	return exists, nil
}

func newTestUserService(t *testing.T) (*UserService, *fakeUserRepository, *fakeBlacklist) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	users := &fakeUserRepository{users: map[string]domain.User{
		"rv32_author": {ID: 3, Login: "rv32_author", PasswordHash: string(hash), Role: domain.UserRoleUser},
	}}
	blacklist := &fakeBlacklist{revoked: map[string]time.Duration{}}
	return NewUserService(users, blacklist, []byte(strings.Repeat("s", 32)), time.Hour), users, blacklist
}

func TestUserServiceHashesRegistrationPassword(t *testing.T) {
	service, users, _ := newTestUserService(t)
	if _, err := service.Register(context.Background(), "rv32_new", "example-password-for-test"); err != nil {
		t.Fatal(err)
	}
	if users.registered != "rv32_new" || users.passwordHash == "example-password-for-test" {
		t.Fatal("логин или хеш пароля сохранён некорректно")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(users.passwordHash), []byte("example-password-for-test")); err != nil {
		t.Fatalf("хеш пароля не соответствует входным данным: %v", err)
	}
	if _, err := service.Register(context.Background(), "x", "short"); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("неверная регистрация принята: %v", err)
	}
}

func TestUserServiceIssuesAndParsesAccessToken(t *testing.T) {
	service, _, _ := newTestUserService(t)
	token, err := service.Authenticate(context.Background(), "rv32_author", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if token.Value == "" || token.ExpiresIn != time.Hour || token.User.PasswordHash != "" {
		t.Fatalf("некорректный токен доступа: %+v", token)
	}
	user, err := service.ParseToken(context.Background(), token.Value)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 3 || user.Login != "rv32_author" || user.Role != domain.UserRoleUser || user.TokenID == "" {
		t.Fatalf("пользователь из JWT определён неверно: %+v", user)
	}
}

func TestUserServiceRejectsWrongCredentials(t *testing.T) {
	service, _, _ := newTestUserService(t)
	for _, login := range []string{"rv32_author", "unknown_user"} {
		if _, err := service.Authenticate(context.Background(), login, "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("логин %s с неверным паролем: %v", login, err)
		}
	}
}

func TestUserServiceRejectsInvalidTokens(t *testing.T) {
	service, _, _ := newTestUserService(t)
	token, err := service.Authenticate(context.Background(), "rv32_author", testPassword)
	if err != nil {
		t.Fatal(err)
	}

	foreign := NewUserService(nil, &fakeBlacklist{revoked: map[string]time.Duration{}}, []byte(strings.Repeat("x", 32)), time.Hour)
	if _, err := foreign.ParseToken(context.Background(), token.Value); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("токен с чужой подписью принят: %v", err)
	}

	expired := *service
	expired.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, err := expired.ParseToken(context.Background(), token.Value); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("просроченный токен принят: %v", err)
	}

	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "3", "role": "user"}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ParseToken(context.Background(), unsigned); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("токен без подписи принят: %v", err)
	}
	if _, err := service.ParseToken(context.Background(), "not-a-jwt"); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("строка вместо JWT принята: %v", err)
	}
}

func TestUserServiceLogoutRevokesToken(t *testing.T) {
	service, _, blacklist := newTestUserService(t)
	token, err := service.Authenticate(context.Background(), "rv32_author", testPassword)
	if err != nil {
		t.Fatal(err)
	}
	user, err := service.ParseToken(context.Background(), token.Value)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Logout(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	if ttl := blacklist.revoked[user.TokenID]; ttl <= 0 || ttl > time.Hour {
		t.Fatalf("время хранения в blacklist: %v", ttl)
	}
	if _, err := service.ParseToken(context.Background(), token.Value); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("отозванный токен принят: %v", err)
	}

	blacklist.err = errors.New("Redis недоступен")
	if _, err := service.ParseToken(context.Background(), token.Value); err == nil || errors.Is(err, ErrUnauthorized) {
		t.Fatalf("ошибка Redis должна отличаться от недействительного токена: %v", err)
	}
}
