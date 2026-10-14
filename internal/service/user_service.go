package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

var (
	ErrInvalidUser        = errors.New("логин или пароль заполнены некорректно")
	ErrLoginAlreadyExists = errors.New("пользователь с таким логином уже существует")
	ErrInvalidCredentials = errors.New("неверный логин или пароль")
	ErrUnauthorized       = errors.New("пользователь не аутентифицирован")
)

var loginPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,63}$`)

// jwtIssuer указывается в JWT и проверяется при разборе токена
const jwtIssuer = "riscv-lab"

// dummyPasswordHash сравнивается с паролем для несуществующего логина,
// чтобы время ответа не раскрывало наличие пользователя
var dummyPasswordHash = sync.OnceValue(func() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	return hash
})

type UserRepository interface {
	RegisterUser(ctx context.Context, login, passwordHash string) (int64, error)
	GetUserByLogin(ctx context.Context, login string) (domain.User, error)
}

// TokenBlacklist хранит идентификаторы отозванных JWT
type TokenBlacklist interface {
	Add(ctx context.Context, tokenID, login string, ttl time.Duration) error
	Contains(ctx context.Context, tokenID string) (bool, error)
}

// accessClaims описывает полезную нагрузку JWT доступа
type accessClaims struct {
	jwt.RegisteredClaims
	Login string          `json:"login"`
	Role  domain.UserRole `json:"role"`
}

type UserService struct {
	repository UserRepository
	blacklist  TokenBlacklist
	secret     []byte
	ttl        time.Duration
	now        func() time.Time
}

func NewUserService(repository UserRepository, blacklist TokenBlacklist, secret []byte, ttl time.Duration) *UserService {
	return &UserService{repository: repository, blacklist: blacklist, secret: secret, ttl: ttl, now: time.Now}
}

func (service *UserService) Register(ctx context.Context, login, password string) (int64, error) {
	login = strings.TrimSpace(login)
	if !loginPattern.MatchString(login) || len(password) < 8 || len(password) > 72 {
		return 0, ErrInvalidUser
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	id, err := service.repository.RegisterUser(ctx, login, string(hash))
	if errors.Is(err, repository.ErrLoginAlreadyExists) {
		return 0, ErrLoginAlreadyExists
	}
	return id, err
}

// Authenticate проверяет логин и пароль и выпускает JWT доступа
func (service *UserService) Authenticate(ctx context.Context, login, password string) (domain.AccessToken, error) {
	user, err := service.repository.GetUserByLogin(ctx, strings.TrimSpace(login))
	if errors.Is(err, repository.ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(password))
		return domain.AccessToken{}, ErrInvalidCredentials
	}
	if err != nil {
		return domain.AccessToken{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return domain.AccessToken{}, ErrInvalidCredentials
	}

	issuedAt := service.now()
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   strconv.FormatInt(user.ID, 10),
			Issuer:    jwtIssuer,
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(service.ttl)),
		},
		Login: user.Login,
		Role:  user.Role,
	}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(service.secret)
	if err != nil {
		return domain.AccessToken{}, fmt.Errorf("подпись JWT: %w", err)
	}
	user.PasswordHash = ""
	return domain.AccessToken{Value: value, ExpiresIn: service.ttl, User: user}, nil
}

// ParseToken проверяет подпись, срок действия и отсутствие JWT в blacklist
func (service *UserService) ParseToken(ctx context.Context, value string) (domain.AuthenticatedUser, error) {
	claims := &accessClaims{}
	_, err := jwt.ParseWithClaims(value, claims, func(*jwt.Token) (any, error) {
		return service.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(jwtIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(service.now),
	)
	if err != nil {
		return domain.AuthenticatedUser{}, ErrUnauthorized
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || id <= 0 || claims.ID == "" {
		return domain.AuthenticatedUser{}, ErrUnauthorized
	}
	if claims.Role != domain.UserRoleUser && claims.Role != domain.UserRoleModerator {
		return domain.AuthenticatedUser{}, ErrUnauthorized
	}
	revoked, err := service.blacklist.Contains(ctx, claims.ID)
	if err != nil {
		return domain.AuthenticatedUser{}, err
	}
	if revoked {
		return domain.AuthenticatedUser{}, ErrUnauthorized
	}
	return domain.AuthenticatedUser{
		ID: id, Login: claims.Login, Role: claims.Role,
		TokenID: claims.ID, ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

// Logout отзывает JWT до окончания срока его действия
func (service *UserService) Logout(ctx context.Context, user domain.AuthenticatedUser) error {
	if user.TokenID == "" {
		return ErrUnauthorized
	}
	return service.blacklist.Add(ctx, user.TokenID, user.Login, user.ExpiresAt.Sub(service.now()))
}
