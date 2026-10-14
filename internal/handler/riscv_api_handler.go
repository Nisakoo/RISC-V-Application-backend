package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/middleware"
	"riscv-lab/internal/service"
	"riscv-lab/internal/storage"
)

type apiMedia interface {
	mediaURLProvider
	Upload(ctx context.Context, kind string, header *multipart.FileHeader) (string, error)
	Remove(ctx context.Context, key string) error
}

type RiscVAPIHandler struct {
	service *service.RiscVAPIService
	media   apiMedia
}

func NewRiscVAPIHandler(apiService *service.RiscVAPIService, media apiMedia) *RiscVAPIHandler {
	return &RiscVAPIHandler{service: apiService, media: media}
}

type apiCommand struct {
	ID           int64                     `json:"id"`
	Mnemonic     string                    `json:"mnemonic"`
	Description  string                    `json:"description,omitempty"`
	OperandCount int                       `json:"operand_count"`
	LengthBits   int                       `json:"length_bits"`
	ImageKey     string                    `json:"image_key,omitempty"`
	VideoKey     string                    `json:"video_key,omitempty"`
	ImageURL     string                    `json:"image_url"`
	VideoURL     string                    `json:"video_url"`
	Status       domain.RiscVCommandStatus `json:"status"`
	CreatedAt    time.Time                 `json:"created_at"`
	PublishedAt  *time.Time                `json:"published_at,omitempty"`
	LikeCount    int64                     `json:"like_count"`
	CreatorLogin string                    `json:"creator_login"`
}

// apiListCommand дополняет команду списка признаком авторства текущего пользователя
type apiListCommand struct {
	apiCommand
	IsCreator int `json:"is_creator"`
}

// apiFeedCommand дополняет команду ленты признаком лайка текущего пользователя и идентификатором следующей команды
type apiFeedCommand struct {
	apiCommand
	IsLiked int   `json:"is_liked"`
	NextID  int64 `json:"next_id"`
}

// apiFlag переводит логический признак в значение 0/1
func apiFlag(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (handler *RiscVAPIHandler) serialize(ctx context.Context, command domain.RiscVCommand) apiCommand {
	return apiCommand{
		ID: command.ID, Mnemonic: command.Mnemonic, Description: command.Description,
		OperandCount: command.OperandCount, LengthBits: command.LengthBits,
		ImageKey: command.ImageKey, VideoKey: command.VideoKey,
		ImageURL: handler.media.ImageURL(ctx, command.ImageKey),
		VideoURL: handler.media.VideoURL(ctx, command.VideoKey),
		Status:   command.Status, CreatedAt: command.CreatedAt, PublishedAt: command.PublishedAt,
		LikeCount: command.LikeCount, CreatorLogin: command.CreatorLogin,
	}
}

// apiErrorResponse описывает ответ с ошибкой
type apiErrorResponse struct {
	Message string `json:"message" example:"Команда RISC-V не найдена."`
}

// apiLikeRequest описывает установку (1) или отмену (0) лайка
type apiLikeRequest struct {
	Like *int `json:"like" enums:"0,1" example:"1"`
}

// apiLikeResponse описывает результат изменения лайка
type apiLikeResponse struct {
	Like      int   `json:"like" example:"1"`
	LikeCount int64 `json:"like_count" example:"4"`
}

func apiFailure(context *gin.Context, status int, message string) {
	context.JSON(status, apiErrorResponse{Message: message})
}

func apiInternalError(context *gin.Context, operation string, err error) {
	log.Printf("%s: %v", operation, err)
	apiFailure(context, http.StatusInternalServerError, "Не удалось выполнить операцию.")
}

func apiID(context *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(context.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		apiFailure(context, http.StatusBadRequest, "Некорректный идентификатор команды RISC-V.")
		return 0, false
	}
	return id, true
}

func apiOperandCount(context *gin.Context, name string, fallback int) (int, bool) {
	value := context.Query(name)
	if value == "" {
		return fallback, true
	}
	count, err := strconv.Atoi(value)
	if err != nil || count < 0 || count > 3 {
		apiFailure(context, http.StatusBadRequest, "Количество операндов должно быть от 0 до 3.")
		return 0, false
	}
	return count, true
}

// List godoc
// @Summary      Список опубликованных команд RISC-V
// @Description  Возвращает опубликованные команды с фильтрацией по количеству операндов. Доступен гостю; для аутентифицированного пользователя is_creator = 1 у его команд.
// @Tags         riscv_commands
// @Produce      json
// @Param        min_operands  query     int  false  "Минимальное количество операндов"   minimum(0) maximum(3)
// @Param        max_operands  query     int  false  "Максимальное количество операндов"  minimum(0) maximum(3)
// @Success      200           {array}   apiListCommand
// @Failure      400           {object}  apiErrorResponse
// @Failure      500           {object}  apiErrorResponse
// @Security     BearerAuth
// @Router       /api/riscv_commands [get]
func (handler *RiscVAPIHandler) List(context *gin.Context) {
	minimum, ok := apiOperandCount(context, "min_operands", 0)
	if !ok {
		return
	}
	maximum, ok := apiOperandCount(context, "max_operands", 3)
	if !ok {
		return
	}
	commands, err := handler.service.List(context.Request.Context(), minimum, maximum)
	if err != nil {
		apiInternalError(context, "список команд RISC-V", err)
		return
	}
	userID := middleware.CurrentUserID(context)
	items := make([]apiListCommand, 0, len(commands))
	for _, command := range commands {
		items = append(items, apiListCommand{
			apiCommand: handler.serialize(context.Request.Context(), command),
			IsCreator:  apiFlag(userID > 0 && command.CreatorID == userID),
		})
	}
	context.JSON(http.StatusOK, items)
}

// Feed godoc
// @Summary      Лента опубликованных команд RISC-V
// @Description  Без id возвращает первую команду ленты, с id — указанную, с id и next=true — следующую после неё. Доступна гостю; is_liked = 1, если аутентифицированный пользователь поставил лайк.
// @Tags         riscv_commands
// @Produce      json
// @Param        id    query     int   false  "Идентификатор команды RISC-V"  minimum(1)
// @Param        next  query     bool  false  "Вернуть следующую команду после id"
// @Success      200   {object}  apiFeedCommand
// @Failure      400   {object}  apiErrorResponse
// @Failure      404   {object}  apiErrorResponse
// @Failure      500   {object}  apiErrorResponse
// @Security     BearerAuth
// @Router       /api/riscv_commands/feed [get]
func (handler *RiscVAPIHandler) Feed(context *gin.Context) {
	var id *int64
	if value := context.Query("id"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			apiFailure(context, http.StatusBadRequest, "Некорректный идентификатор команды RISC-V.")
			return
		}
		id = &parsed
	}
	next := false
	if value := context.Query("next"); value != "" {
		var err error
		next, err = strconv.ParseBool(value)
		if err != nil || (next && id == nil) {
			apiFailure(context, http.StatusBadRequest, "Для перехода к следующей команде укажите id и next=true.")
			return
		}
	}
	command, following, liked, err := handler.service.Feed(context.Request.Context(), middleware.CurrentUserID(context), id, next)
	if errors.Is(err, service.ErrRiscVCommandNotFound) {
		apiFailure(context, http.StatusNotFound, "Команда RISC-V не найдена.")
		return
	}
	if err != nil {
		apiInternalError(context, "лента команд RISC-V", err)
		return
	}
	context.JSON(http.StatusOK, apiFeedCommand{
		apiCommand: handler.serialize(context.Request.Context(), command),
		IsLiked:    apiFlag(liked),
		NextID:     following.ID,
	})
}

// Draft godoc
// @Summary      Черновик команды RISC-V текущего пользователя
// @Description  Возвращает единственный черновик пользователя, определённого по JWT; идентификатор не передаётся.
// @Tags         riscv_commands
// @Produce      json
// @Success      200  {object}  apiCommand
// @Failure      401  {object}  apiErrorResponse
// @Failure      403  {object}  apiErrorResponse
// @Failure      404  {object}  apiErrorResponse
// @Failure      500  {object}  apiErrorResponse
// @Security     BearerAuth
// @Router       /api/riscv_commands/draft [get]
func (handler *RiscVAPIHandler) Draft(context *gin.Context) {
	command, err := handler.service.Draft(context.Request.Context(), middleware.CurrentUserID(context))
	if errors.Is(err, service.ErrRiscVCommandNotFound) {
		apiFailure(context, http.StatusNotFound, "Черновик команды RISC-V не найден.")
		return
	}
	if err != nil {
		apiInternalError(context, "черновик команды RISC-V", err)
		return
	}
	context.JSON(http.StatusOK, handler.serialize(context.Request.Context(), command))
}

// Create godoc
// @Summary      Создание черновика команды RISC-V
// @Description  Создаёт черновик из мнемоники, изображения и короткого видео. Файлы сохраняются в MinIO под латинскими именами, создатель определяется по JWT.
// @Tags         riscv_commands
// @Accept       mpfd
// @Produce      json
// @Param        mnemonic  formData  string  true  "Мнемоника команды RISC-V"  example(XORI)
// @Param        image     formData  file    true  "Изображение PNG, JPEG или WebP до 5 МиБ"
// @Param        video     formData  file    true  "Короткое видео MP4 до 50 МиБ"
// @Success      201       {object}  apiCommand
// @Failure      400       {object}  apiErrorResponse
// @Failure      401       {object}  apiErrorResponse
// @Failure      403       {object}  apiErrorResponse
// @Failure      409       {object}  apiErrorResponse
// @Failure      500       {object}  apiErrorResponse
// @Security     BearerAuth
// @Router       /api/riscv_commands [post]
func (handler *RiscVAPIHandler) Create(context *gin.Context) {
	context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, 56<<20)
	if err := context.Request.ParseMultipartForm(8 << 20); err != nil {
		apiFailure(context, http.StatusBadRequest, "Некорректная форма команды RISC-V.")
		return
	}
	for name := range context.Request.MultipartForm.Value {
		if name != "mnemonic" {
			apiFailure(context, http.StatusBadRequest, "Форма содержит недопустимые поля.")
			return
		}
	}
	for name := range context.Request.MultipartForm.File {
		if name != "image" && name != "video" {
			apiFailure(context, http.StatusBadRequest, "Форма содержит недопустимые файлы.")
			return
		}
	}
	image, imageError := context.FormFile("image")
	video, videoError := context.FormFile("video")
	if imageError != nil || videoError != nil {
		apiFailure(context, http.StatusBadRequest, "Укажите изображение и короткое видео команды RISC-V.")
		return
	}
	ctx := context.Request.Context()
	imageKey, err := handler.media.Upload(ctx, "images", image)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidMedia) {
			apiFailure(context, http.StatusBadRequest, "Некорректное изображение команды RISC-V.")
		} else {
			apiInternalError(context, "загрузка изображения команды RISC-V", err)
		}
		return
	}
	videoKey, err := handler.media.Upload(ctx, "video", video)
	if err != nil {
		_ = handler.media.Remove(ctx, imageKey)
		if errors.Is(err, storage.ErrInvalidMedia) {
			apiFailure(context, http.StatusBadRequest, "Некорректное видео команды RISC-V.")
		} else {
			apiInternalError(context, "загрузка видео команды RISC-V", err)
		}
		return
	}
	command, err := handler.service.Create(ctx, middleware.CurrentUserID(context), context.PostForm("mnemonic"), imageKey, videoKey)
	if err != nil {
		_ = handler.media.Remove(ctx, imageKey)
		_ = handler.media.Remove(ctx, videoKey)
		if errors.Is(err, service.ErrInvalidRiscVCommand) {
			apiFailure(context, http.StatusBadRequest, "Некорректная мнемоника команды RISC-V.")
			return
		}
		if errors.Is(err, service.ErrDraftAlreadyExists) {
			apiFailure(context, http.StatusConflict, "У пользователя уже есть черновик команды RISC-V.")
			return
		}
		apiInternalError(context, "создание команды RISC-V", err)
		return
	}
	context.Header("Location", "/api/riscv_commands/draft")
	context.JSON(http.StatusCreated, handler.serialize(ctx, command))
}

func readAPIJSON(context *gin.Context, target any) error {
	context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, 64<<10)
	decoder := json.NewDecoder(context.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("тело запроса должно содержать один объект JSON")
	}
	return nil
}

// Publish godoc
// @Summary      Публикация черновика команды RISC-V
// @Description  Заполняет описание и поля команды и меняет статус черновика текущего пользователя на published. Системные поля передавать запрещено.
// @Tags         riscv_commands
// @Accept       json
// @Produce      json
// @Param        id    path      int                              true  "Идентификатор черновика"  minimum(1)
// @Param        body  body      domain.PublishRiscVCommandDraft  true  "Поля публикации"
// @Success      200   {object}  apiCommand
// @Failure      400   {object}  apiErrorResponse
// @Failure      401   {object}  apiErrorResponse
// @Failure      403   {object}  apiErrorResponse
// @Failure      404   {object}  apiErrorResponse
// @Failure      500   {object}  apiErrorResponse
// @Security     BearerAuth
// @Router       /api/riscv_commands/{id}/publish [put]
func (handler *RiscVAPIHandler) Publish(context *gin.Context) {
	id, ok := apiID(context)
	if !ok {
		return
	}
	var input domain.PublishRiscVCommandDraft
	if err := readAPIJSON(context, &input); err != nil {
		apiFailure(context, http.StatusBadRequest, "Некорректные поля публикации команды RISC-V.")
		return
	}
	command, err := handler.service.Publish(context.Request.Context(), middleware.CurrentUserID(context), id, input)
	if errors.Is(err, service.ErrInvalidRiscVCommand) {
		apiFailure(context, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, service.ErrRiscVCommandNotFound) {
		apiFailure(context, http.StatusNotFound, "Черновик команды RISC-V не найден.")
		return
	}
	if err != nil {
		apiInternalError(context, "публикация команды RISC-V", err)
		return
	}
	context.JSON(http.StatusOK, handler.serialize(context.Request.Context(), command))
}

// Delete godoc
// @Summary      Логическое удаление команды RISC-V
// @Description  Меняет статус собственной команды пользователя на deleted.
// @Tags         riscv_commands
// @Produce      json
// @Param        id   path      int  true  "Идентификатор команды"  minimum(1)
// @Success      204
// @Failure      400  {object}  apiErrorResponse
// @Failure      401  {object}  apiErrorResponse
// @Failure      403  {object}  apiErrorResponse
// @Failure      404  {object}  apiErrorResponse
// @Failure      500  {object}  apiErrorResponse
// @Security     BearerAuth
// @Router       /api/riscv_commands/{id} [delete]
func (handler *RiscVAPIHandler) Delete(context *gin.Context) {
	id, ok := apiID(context)
	if !ok {
		return
	}
	if err := handler.service.Delete(context.Request.Context(), middleware.CurrentUserID(context), id); err != nil {
		if errors.Is(err, service.ErrRiscVCommandNotFound) {
			apiFailure(context, http.StatusNotFound, "Команда RISC-V не найдена среди команд пользователя.")
			return
		}
		apiInternalError(context, "удаление команды RISC-V", err)
		return
	}
	context.Status(http.StatusNoContent)
}

// Like godoc
// @Summary      Лайк опубликованной команды RISC-V
// @Description  like = 1 ставит лайк от текущего пользователя, like = 0 отменяет его.
// @Tags         riscv_commands
// @Accept       json
// @Produce      json
// @Param        id    path      int             true  "Идентификатор команды"  minimum(1)
// @Param        body  body      apiLikeRequest  true  "Признак лайка 0/1"
// @Success      200   {object}  apiLikeResponse
// @Failure      400   {object}  apiErrorResponse
// @Failure      401   {object}  apiErrorResponse
// @Failure      403   {object}  apiErrorResponse
// @Failure      404   {object}  apiErrorResponse
// @Failure      500   {object}  apiErrorResponse
// @Security     BearerAuth
// @Router       /api/riscv_commands/{id}/like [post]
func (handler *RiscVAPIHandler) Like(context *gin.Context) {
	id, ok := apiID(context)
	if !ok {
		return
	}
	var input apiLikeRequest
	if err := readAPIJSON(context, &input); err != nil || input.Like == nil || (*input.Like != 0 && *input.Like != 1) {
		apiFailure(context, http.StatusBadRequest, "Поле like должно равняться 0 или 1.")
		return
	}
	count, err := handler.service.Like(context.Request.Context(), middleware.CurrentUserID(context), id, *input.Like == 1)
	if errors.Is(err, service.ErrRiscVCommandNotFound) {
		apiFailure(context, http.StatusNotFound, "Опубликованная команда RISC-V не найдена.")
		return
	}
	if err != nil {
		apiInternalError(context, "лайк команды RISC-V", err)
		return
	}
	context.JSON(http.StatusOK, apiLikeResponse{Like: *input.Like, LikeCount: count})
}
