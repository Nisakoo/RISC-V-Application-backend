package handler

import (
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/route"
	"riscv-lab/internal/service"
)

const (
	currentModeratorID      int64 = 1
	maximumRequestBodyBytes       = 522 * 1024 * 1024
)

type mediaURLProvider interface {
	URL(key string) string
}

type RiscVCommandHandler struct {
	service *service.RiscVCommandService
	media   mediaURLProvider
}

func NewRiscVCommandHandler(riscvCommandService *service.RiscVCommandService, media mediaURLProvider) *RiscVCommandHandler {
	return &RiscVCommandHandler{service: riscvCommandService, media: media}
}

func (handler *RiscVCommandHandler) Feed(context *gin.Context) {
	idParameter := context.Query("id")
	var id *int64
	if idParameter != "" {
		parsedID, conversionError := strconv.ParseInt(idParameter, 10, 64)
		if conversionError != nil || parsedID <= 0 {
			renderError(context, http.StatusBadRequest, newErrorPageView(
				"Некорректный идентификатор", "Некорректный идентификатор команды.", activePageFeed, pageClassFeed, route.Feed, "Открыть ленту",
			))
			return
		}
		id = &parsedID
	}

	command, nextCommand, err := handler.service.GetFeedRiscVCommand(context.Request.Context(), id)
	if err != nil {
		if !errors.Is(err, service.ErrRiscVCommandNotFound) {
			log.Printf("получение команды ленты RISC-V: %v", err)
			renderInternalError(context, activePageFeed)
			return
		}
		message := "Команда не найдена."
		if id == nil {
			message = "В ленте пока нет доступных команд."
		}
		renderError(context, http.StatusNotFound, newErrorPageView(
			"Команда не найдена", message, activePageFeed, pageClassFeed, route.Feed, "Открыть ленту",
		))
		return
	}

	context.HTML(http.StatusOK, "feed.html", feedPageView{
		Layout:              newLayoutView(command.Mnemonic+" · RISC-V", activePageFeed),
		RiscVCommand:        handler.toRiscVCommandView(command),
		NextRiscVCommandURL: route.Feed + "?id=" + strconv.FormatInt(nextCommand.ID, 10),
	})
}

func (handler *RiscVCommandHandler) AddPage(context *gin.Context) {
	draft, err := handler.service.GetDraft(context.Request.Context(), currentModeratorID)
	if errors.Is(err, service.ErrRiscVCommandNotFound) {
		handler.renderDraft(context, http.StatusOK, domain.RiscVCommand{}, "")
		return
	}
	if err != nil {
		log.Printf("получение черновика команды RISC-V: %v", err)
		renderInternalError(context, activePageDraft)
		return
	}
	handler.renderDraft(context, http.StatusOK, draft, "")
}

func (handler *RiscVCommandHandler) CreateDraft(context *gin.Context) {
	context.Request.Body = http.MaxBytesReader(context.Writer, context.Request.Body, maximumRequestBodyBytes)
	image, imageError := context.FormFile("riscv_command_image")
	video, videoError := context.FormFile("riscv_command_video")
	if imageError != nil || videoError != nil {
		handler.renderDraft(context, http.StatusBadRequest, domain.RiscVCommand{Mnemonic: strings.TrimSpace(context.PostForm("mnemonic"))}, "Выберите изображение и видео команды.")
		return
	}

	imageFile, err := image.Open()
	if err != nil {
		handler.renderDraft(context, http.StatusBadRequest, domain.RiscVCommand{}, "Не удалось прочитать изображение.")
		return
	}
	defer imageFile.Close()
	videoFile, err := video.Open()
	if err != nil {
		handler.renderDraft(context, http.StatusBadRequest, domain.RiscVCommand{}, "Не удалось прочитать видео.")
		return
	}
	defer videoFile.Close()

	input := domain.CreateRiscVCommandDraft{
		Mnemonic: context.PostForm("mnemonic"),
		Image:    uploadedMedia(image, imageFile),
		Video:    uploadedMedia(video, videoFile),
	}
	_, err = handler.service.CreateDraft(context.Request.Context(), currentModeratorID, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRiscVCommand) || errors.Is(err, service.ErrInvalidImage) || errors.Is(err, service.ErrInvalidVideo) || errors.Is(err, service.ErrDraftAlreadyExists) {
			handler.renderDraft(context, http.StatusBadRequest, domain.RiscVCommand{Mnemonic: strings.ToUpper(strings.TrimSpace(input.Mnemonic))}, err.Error())
			return
		}
		log.Printf("создание черновика команды RISC-V: %v", err)
		renderInternalError(context, activePageDraft)
		return
	}
	context.Redirect(http.StatusSeeOther, route.Add)
}

func (handler *RiscVCommandHandler) PublishDraft(context *gin.Context) {
	input, err := publicationInput(context)
	if err != nil {
		handler.renderCurrentDraftWithError(context, http.StatusBadRequest, err.Error())
		return
	}
	command, err := handler.service.PublishDraft(context.Request.Context(), currentModeratorID, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRiscVCommand) || errors.Is(err, service.ErrRiscVCommandNotFound) {
			handler.renderCurrentDraftWithError(context, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("публикация команды RISC-V: %v", err)
		renderInternalError(context, activePageDraft)
		return
	}
	context.Redirect(http.StatusSeeOther, route.Feed+"?id="+strconv.FormatInt(command.ID, 10))
}

func (handler *RiscVCommandHandler) Grid(context *gin.Context) {
	minimumOperandCount := operandCountParameter(context, "min_operands", 0)
	maximumOperandCount := operandCountParameter(context, "max_operands", 3)
	commands, minimumOperandCount, maximumOperandCount, err := handler.service.GetVisibleByOperandRange(
		context.Request.Context(), minimumOperandCount, maximumOperandCount,
	)
	if err != nil {
		log.Printf("получение списка команд RISC-V: %v", err)
		renderInternalError(context, activePageGrid)
		return
	}
	views := make([]riscvCommandView, 0, len(commands))
	for _, command := range commands {
		views = append(views, handler.toRiscVCommandView(command))
	}

	context.HTML(http.StatusOK, "grid.html", gridPageView{
		Layout:              newLayoutView("Команды · RISC-V", activePageGrid),
		RiscVCommands:       views,
		MinimumOperandCount: minimumOperandCount,
		MaximumOperandCount: maximumOperandCount,
		RangeStart:          minimumOperandCount * 100 / 3,
		RangeEnd:            maximumOperandCount * 100 / 3,
	})
}

func (handler *RiscVCommandHandler) Delete(context *gin.Context) {
	id, err := strconv.ParseInt(context.PostForm("riscv_command_id"), 10, 64)
	if err != nil || id <= 0 {
		renderError(context, http.StatusBadRequest, newErrorPageView(
			"Некорректный идентификатор", "Некорректный идентификатор команды.", activePageGrid, pageClassPlain, route.Grid, "Открыть команды",
		))
		return
	}
	if err := handler.service.Delete(context.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrRiscVCommandNotFound) {
			renderError(context, http.StatusNotFound, newErrorPageView(
				"Команда не найдена", "Удалить команду не удалось: она не существует или уже удалена.", activePageGrid, pageClassPlain, route.Grid, "Открыть команды",
			))
			return
		}
		log.Printf("логическое удаление команды RISC-V: %v", err)
		renderInternalError(context, activePageGrid)
		return
	}
	context.Redirect(http.StatusSeeOther, route.Grid)
}

func operandCountParameter(context *gin.Context, name string, fallback int) int {
	value, err := strconv.Atoi(context.Query(name))
	if err != nil || value < 0 || value > 3 {
		return fallback
	}
	return value
}

func publicationInput(context *gin.Context) (domain.PublishRiscVCommandDraft, error) {
	operandCount, err := strconv.Atoi(context.PostForm("operand_count"))
	if err != nil {
		return domain.PublishRiscVCommandDraft{}, errors.New("количество операндов должно быть числом")
	}
	lengthBits, err := strconv.Atoi(context.PostForm("length_bits"))
	if err != nil {
		return domain.PublishRiscVCommandDraft{}, errors.New("длина команды должна быть числом")
	}
	return domain.PublishRiscVCommandDraft{
		AssemblyExample: context.PostForm("assembly_example"),
		Description:     context.PostForm("description"),
		Format:          domain.InstructionFormat(context.PostForm("format")),
		OperandCount:    operandCount,
		LengthBits:      lengthBits,
	}, nil
}

func uploadedMedia(header *multipart.FileHeader, file multipart.File) domain.UploadedMedia {
	return domain.UploadedMedia{
		OriginalName: header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Size:         header.Size,
		Reader:       file,
	}
}

func (handler *RiscVCommandHandler) renderDraft(context *gin.Context, status int, command domain.RiscVCommand, message string) {
	context.HTML(status, "draft.html", draftPageView{
		Layout:       newLayoutView("Добавление команды · RISC-V", activePageDraft),
		HasDraft:     command.ID > 0,
		RiscVCommand: handler.toRiscVCommandView(command),
		ErrorMessage: message,
	})
}

func (handler *RiscVCommandHandler) renderCurrentDraftWithError(context *gin.Context, status int, message string) {
	draft, err := handler.service.GetDraft(context.Request.Context(), currentModeratorID)
	if err != nil {
		renderError(context, http.StatusNotFound, newErrorPageView(
			"Черновик не найден", "Сначала создайте черновик команды.", activePageDraft, pageClassPlain, route.Add, "Создать черновик",
		))
		return
	}
	handler.renderDraft(context, status, draft, message)
}

func (handler *RiscVCommandHandler) toRiscVCommandView(command domain.RiscVCommand) riscvCommandView {
	return riscvCommandView{
		ID:              command.ID,
		Mnemonic:        command.Mnemonic,
		AssemblyExample: command.AssemblyExample,
		Description:     command.Description,
		Format:          command.Format,
		OperandCount:    command.OperandCount,
		LengthBits:      command.LengthBits,
		ImageURL:        handler.media.URL(command.ImageKey),
		VideoURL:        handler.media.URL(command.VideoKey),
		LikeCount:       command.LikeCount,
	}
}

func renderInternalError(context *gin.Context, activePage activePage) {
	renderError(context, http.StatusInternalServerError, newErrorPageView(
		"Ошибка сервера", "Не удалось выполнить операцию. Повторите попытку позже.", activePage, pageClassPlain, route.Grid, "Открыть команды",
	))
}

func renderError(context *gin.Context, status int, view errorPageView) {
	context.HTML(status, "error.html", view)
}
