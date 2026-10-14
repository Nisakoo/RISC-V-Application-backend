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
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
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

type apiCreator struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
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
	IsCreator    int                       `json:"is_creator"`
	Creator      apiCreator                `json:"creator"`
}

// apiFeedCommand дополняет команду ленты идентификатором следующей команды
type apiFeedCommand struct {
	apiCommand
	NextID int64 `json:"next_id"`
}

func (handler *RiscVAPIHandler) serialize(ctx context.Context, command domain.RiscVCommand) apiCommand {
	isCreator := 0
	if command.CreatorID == service.CurrentUserID() {
		isCreator = 1
	}
	return apiCommand{
		ID: command.ID, Mnemonic: command.Mnemonic, Description: command.Description,
		OperandCount: command.OperandCount, LengthBits: command.LengthBits,
		ImageKey: command.ImageKey, VideoKey: command.VideoKey,
		ImageURL: handler.media.ImageURL(ctx, command.ImageKey),
		VideoURL: handler.media.VideoURL(ctx, command.VideoKey),
		Status:   command.Status, CreatedAt: command.CreatedAt, PublishedAt: command.PublishedAt,
		LikeCount: command.LikeCount, IsCreator: isCreator,
		Creator: apiCreator{ID: command.CreatorID, Login: command.CreatorLogin},
	}
}

func apiFailure(context *gin.Context, status int, message string) {
	context.JSON(status, gin.H{"message": message})
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
	items := make([]apiCommand, 0, len(commands))
	for _, command := range commands {
		items = append(items, handler.serialize(context.Request.Context(), command))
	}
	context.JSON(http.StatusOK, items)
}

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
	command, following, err := handler.service.Feed(context.Request.Context(), id, next)
	if errors.Is(err, service.ErrRiscVCommandNotFound) {
		apiFailure(context, http.StatusNotFound, "Команда RISC-V не найдена.")
		return
	}
	if err != nil {
		apiInternalError(context, "лента команд RISC-V", err)
		return
	}
	context.JSON(http.StatusOK, apiFeedCommand{apiCommand: handler.serialize(context.Request.Context(), command), NextID: following.ID})
}

func (handler *RiscVAPIHandler) Draft(context *gin.Context) {
	command, err := handler.service.Draft(context.Request.Context())
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
	command, err := handler.service.Create(ctx, context.PostForm("mnemonic"), imageKey, videoKey)
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
	command, err := handler.service.Publish(context.Request.Context(), id, input)
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

func (handler *RiscVAPIHandler) Delete(context *gin.Context) {
	id, ok := apiID(context)
	if !ok {
		return
	}
	if err := handler.service.Delete(context.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrRiscVCommandNotFound) {
			apiFailure(context, http.StatusNotFound, "Команда RISC-V не найдена среди команд пользователя.")
			return
		}
		apiInternalError(context, "удаление команды RISC-V", err)
		return
	}
	context.Status(http.StatusNoContent)
}

func (handler *RiscVAPIHandler) Like(context *gin.Context) {
	id, ok := apiID(context)
	if !ok {
		return
	}
	var input struct {
		Like *int `json:"like"`
	}
	if err := readAPIJSON(context, &input); err != nil || input.Like == nil || (*input.Like != 0 && *input.Like != 1) {
		apiFailure(context, http.StatusBadRequest, "Поле like должно равняться 0 или 1.")
		return
	}
	count, err := handler.service.Like(context.Request.Context(), id, *input.Like == 1)
	if errors.Is(err, service.ErrRiscVCommandNotFound) {
		apiFailure(context, http.StatusNotFound, "Опубликованная команда RISC-V не найдена.")
		return
	}
	if err != nil {
		apiInternalError(context, "лайк команды RISC-V", err)
		return
	}
	context.JSON(http.StatusOK, gin.H{"like": *input.Like, "like_count": count})
}

func (handler *RiscVAPIHandler) Register(context *gin.Context) {
	var input struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
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
	context.JSON(http.StatusCreated, gin.H{"id": id, "login": input.Login})
}

func (handler *RiscVAPIHandler) AuthenticationStub(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"message": "Аутентификация будет реализована в лабораторной работе №4."})
}

func (handler *RiscVAPIHandler) LogoutStub(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"message": "Деавторизация будет реализована в лабораторной работе №4."})
}
