package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/service"
)

const (
	bearerPrefix       = "Bearer "
	currentUserContext = "riscv.current_user"
)

// TokenParser проверяет JWT и возвращает пользователя запроса
type TokenParser interface {
	ParseToken(ctx context.Context, value string) (domain.AuthenticatedUser, error)
}

// CurrentUser возвращает пользователя, подтверждённого middleware, или false для гостя
func CurrentUser(context *gin.Context) (domain.AuthenticatedUser, bool) {
	value, exists := context.Get(currentUserContext)
	if !exists {
		return domain.AuthenticatedUser{}, false
	}
	user, ok := value.(domain.AuthenticatedUser)
	return user, ok
}

// CurrentUserID возвращает идентификатор пользователя или 0 для гостя
func CurrentUserID(context *gin.Context) int64 {
	user, _ := CurrentUser(context)
	return user.ID
}

func bearerToken(context *gin.Context) (string, bool) {
	header := context.GetHeader("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(bearerPrefix):])
	return token, token != ""
}

func abort(context *gin.Context, status int, message string) {
	context.AbortWithStatusJSON(status, gin.H{"message": message})
}

// OptionalAuth определяет пользователя по JWT, если токен действителен.
// Публичные методы чтения работают независимо от токена: при его отсутствии
// или недействительности запрос выполняется от имени гостя.
func OptionalAuth(parser TokenParser) gin.HandlerFunc {
	return func(context *gin.Context) {
		if token, ok := bearerToken(context); ok {
			if user, err := parser.ParseToken(context.Request.Context(), token); err == nil {
				context.Set(currentUserContext, user)
			}
		}
		context.Next()
	}
}

// RequireAuth пропускает только аутентифицированных пользователей с одной из указанных ролей.
// Без JWT или с недействительным JWT ответ 401, при недостаточной роли — 403.
func RequireAuth(parser TokenParser, roles ...domain.UserRole) gin.HandlerFunc {
	return func(context *gin.Context) {
		token, ok := bearerToken(context)
		if !ok {
			abort(context, http.StatusUnauthorized, "Требуется аутентификация.")
			return
		}
		user, err := parser.ParseToken(context.Request.Context(), token)
		if errors.Is(err, service.ErrUnauthorized) {
			abort(context, http.StatusUnauthorized, "Токен доступа недействителен или отозван.")
			return
		}
		if err != nil {
			log.Printf("проверка JWT: %v", err)
			abort(context, http.StatusInternalServerError, "Не удалось выполнить операцию.")
			return
		}
		if len(roles) > 0 && !slices.Contains(roles, user.Role) {
			abort(context, http.StatusForbidden, "Недостаточно прав для выполнения действия.")
			return
		}
		context.Set(currentUserContext, user)
		context.Next()
	}
}
