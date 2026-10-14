package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/service"
)

// fakeParser принимает токены user, moderator, revoked и broken
type fakeParser struct{}

func (fakeParser) ParseToken(_ context.Context, value string) (domain.AuthenticatedUser, error) {
	switch value {
	case "user":
		return domain.AuthenticatedUser{ID: 2, Role: domain.UserRoleUser}, nil
	case "moderator":
		return domain.AuthenticatedUser{ID: 1, Role: domain.UserRoleModerator}, nil
	case "broken":
		return domain.AuthenticatedUser{}, errors.New("Redis недоступен")
	default:
		return domain.AuthenticatedUser{}, service.ErrUnauthorized
	}
}

func newRouter(handlers ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers = append(handlers, func(context *gin.Context) {
		context.String(http.StatusOK, strconv.FormatInt(CurrentUserID(context), 10))
	})
	router.GET("/", handlers...)
	return router
}

func request(router *gin.Engine, authorization string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestOptionalAuthTreatsInvalidTokenAsGuest(t *testing.T) {
	router := newRouter(OptionalAuth(fakeParser{}))
	cases := map[string]string{
		"":                 "0",
		"Bearer revoked":   "0",
		"Bearer broken":    "0",
		"Token user":       "0",
		"Bearer user":      "2",
		"Bearer moderator": "1",
	}
	for header, expected := range cases {
		response := request(router, header)
		if response.Code != http.StatusOK || response.Body.String() != expected {
			t.Errorf("заголовок %q: код %d, пользователь %s", header, response.Code, response.Body.String())
		}
	}
}

func TestRequireAuthChecksTokenAndRole(t *testing.T) {
	router := newRouter(RequireAuth(fakeParser{}, domain.UserRoleModerator))
	cases := []struct {
		header string
		status int
	}{
		{"", http.StatusUnauthorized},
		{"Bearer ", http.StatusUnauthorized},
		{"Basic user", http.StatusUnauthorized},
		{"Bearer revoked", http.StatusUnauthorized},
		{"Bearer broken", http.StatusInternalServerError},
		{"Bearer user", http.StatusForbidden},
		{"Bearer moderator", http.StatusOK},
	}
	for _, test := range cases {
		response := request(router, test.header)
		if response.Code != test.status {
			t.Errorf("заголовок %q: ожидался код %d, получен %d", test.header, test.status, response.Code)
		}
		if test.status != http.StatusOK && response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("заголовок %q: ошибка должна возвращаться в JSON", test.header)
		}
	}
}
