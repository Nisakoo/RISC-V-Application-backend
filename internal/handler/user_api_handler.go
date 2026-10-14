package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/middleware"
	"riscv-lab/internal/service"
)

type UserAPIHandler struct {
	service *service.UserService
}

func NewUserAPIHandler(userService *service.UserService) *UserAPIHandler {
	return &UserAPIHandler{service: userService}
}

// apiCredentials описывает логин и пароль пользователя
type apiCredentials struct {
	Login    string `json:"login" example:"rv32_author"`
	Password string `json:"password" example:"correct-horse-battery"`
}

// apiRegisteredUser описывает зарегистрированного пользователя без пароля
type apiRegisteredUser struct {
	ID    int64  `json:"id" example:"7"`
	Login string `json:"login" example:"rv32_author"`
}

// apiAuthentication описывает выпущенный JWT доступа и роль пользователя
type apiAuthentication struct {
	AccessToken string          `json:"access_token"`
	TokenType   string          `json:"token_type" example:"Bearer"`
	ExpiresIn   int64           `json:"expires_in" example:"3600"`
	Login       string          `json:"login" example:"rv32_author"`
	Role        domain.UserRole `json:"role" enums:"user,moderator" example:"user"`
}

// Register godoc
// @Summary      Регистрация пользователя
// @Description  Создаёт пользователя с ролью user. Логин: латиница, цифры и _, 3–64 символа; пароль: 8–72 символа.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      apiCredentials  true  "Логин и пароль"
// @Success      201   {object}  apiRegisteredUser
// @Failure      400   {object}  apiErrorResponse
// @Failure      409   {object}  apiErrorResponse
// @Failure      500   {object}  apiErrorResponse
// @Router       /api/users [post]
func (handler *UserAPIHandler) Register(context *gin.Context) {
	var input apiCredentials
	if err := readAPIJSON(context, &input); err != nil {
		apiFailure(context, http.StatusBadRequest, "Некорректные данные пользователя.")
		return
	}
	input.Login = strings.TrimSpace(input.Login)
	id, err := handler.service.Register(context.Request.Context(), input.Login, input.Password)
	if errors.Is(err, service.ErrInvalidUser) {
		apiFailure(context, http.StatusBadRequest, "Логин или пароль заполнены некорректно.")
		return
	}
	if errors.Is(err, service.ErrLoginAlreadyExists) {
		apiFailure(context, http.StatusConflict, "Пользователь с таким логином уже существует.")
		return
	}
	if err != nil {
		apiInternalError(context, "регистрация пользователя", err)
		return
	}
	context.JSON(http.StatusCreated, apiRegisteredUser{ID: id, Login: input.Login})
}

// Authenticate godoc
// @Summary      Аутентификация пользователя
// @Description  Проверяет логин и пароль и возвращает JWT доступа. Токен передаётся в остальных запросах заголовком Authorization: Bearer <token>.
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      apiCredentials  true  "Логин и пароль"
// @Success      200   {object}  apiAuthentication
// @Failure      400   {object}  apiErrorResponse
// @Failure      401   {object}  apiErrorResponse
// @Failure      500   {object}  apiErrorResponse
// @Router       /api/users/authentication [post]
func (handler *UserAPIHandler) Authenticate(context *gin.Context) {
	var input apiCredentials
	if err := readAPIJSON(context, &input); err != nil || strings.TrimSpace(input.Login) == "" || input.Password == "" {
		apiFailure(context, http.StatusBadRequest, "Укажите логин и пароль.")
		return
	}
	token, err := handler.service.Authenticate(context.Request.Context(), input.Login, input.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		apiFailure(context, http.StatusUnauthorized, "Неверный логин или пароль.")
		return
	}
	if err != nil {
		apiInternalError(context, "аутентификация пользователя", err)
		return
	}
	context.JSON(http.StatusOK, apiAuthentication{
		AccessToken: token.Value,
		TokenType:   "Bearer",
		ExpiresIn:   int64(token.ExpiresIn.Seconds()),
		Login:       token.User.Login,
		Role:        token.User.Role,
	})
}

// Logout godoc
// @Summary      Деавторизация пользователя
// @Description  Помещает текущий JWT в blacklist Redis до истечения его срока; повторно использовать токен нельзя.
// @Tags         users
// @Produce      json
// @Success      204
// @Failure      401  {object}  apiErrorResponse
// @Failure      500  {object}  apiErrorResponse
// @Security     BearerAuth
// @Router       /api/users/logout [post]
func (handler *UserAPIHandler) Logout(context *gin.Context) {
	user, ok := middleware.CurrentUser(context)
	if !ok {
		apiFailure(context, http.StatusUnauthorized, "Требуется аутентификация.")
		return
	}
	if err := handler.service.Logout(context.Request.Context(), user); err != nil {
		apiInternalError(context, "деавторизация пользователя", err)
		return
	}
	context.Status(http.StatusNoContent)
}
