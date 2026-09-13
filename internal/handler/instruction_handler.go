package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/service"
)

type InstructionHandler struct {
	service *service.InstructionService
}

func NewInstructionHandler(instructionService *service.InstructionService) *InstructionHandler {
	return &InstructionHandler{service: instructionService}
}

func (handler *InstructionHandler) Feed(context *gin.Context) {
	idParameter := context.Query("id")
	var id *int
	if idParameter != "" {
		parsedID, conversionError := strconv.Atoi(idParameter)
		if conversionError != nil {
			renderError(context, http.StatusBadRequest, newErrorPageView(
				"Некорректный идентификатор", "Некорректный идентификатор команды.", activePageFeed, pageClassFeed, "/", "Открыть ленту",
			))
			return
		}
		id = &parsedID
	}

	instruction, nextInstruction, err := handler.service.GetFeedInstruction(id)
	if err != nil {
		message := "Команда не найдена."
		if id == nil {
			message = "В ленте пока нет доступных команд."
		}
		renderError(context, http.StatusNotFound, newErrorPageView(
			"Команда не найдена", message, activePageFeed, pageClassFeed, "/", "Открыть ленту",
		))
		return
	}

	context.HTML(http.StatusOK, "feed.html", feedPageView{
		Layout:            newLayoutView(instruction.Mnemonic+" · RISC-V", activePageFeed),
		Instruction:       toInstructionView(instruction),
		NextInstructionID: nextInstruction.ID,
	})
}

func (handler *InstructionHandler) Draft(context *gin.Context) {
	instruction, err := handler.service.GetDraft()
	if err != nil {
		renderError(context, http.StatusNotFound, newErrorPageView(
			"Черновик не найден", "Черновик инструкции не найден.", activePageDraft, pageClassPlain, "/instructions", "Открыть инструкции",
		))
		return
	}

	context.HTML(http.StatusOK, "draft.html", draftPageView{
		Layout:      newLayoutView("Добавление инструкции · RISC-V", activePageDraft),
		Instruction: toInstructionView(instruction),
	})
}

func (handler *InstructionHandler) Grid(context *gin.Context) {
	minimumOperandCount := operandCountParameter(context, "min_operands", 0)
	maximumOperandCount := operandCountParameter(context, "max_operands", 3)
	instructions, minimumOperandCount, maximumOperandCount := handler.service.GetVisibleByOperandRange(minimumOperandCount, maximumOperandCount)
	views := make([]instructionView, 0, len(instructions))
	for _, instruction := range instructions {
		views = append(views, toInstructionView(instruction))
	}

	context.HTML(http.StatusOK, "grid.html", gridPageView{
		Layout:              newLayoutView("Команды · RISC-V", activePageGrid),
		Instructions:        views,
		MinimumOperandCount: minimumOperandCount,
		MaximumOperandCount: maximumOperandCount,
		RangeStart:          minimumOperandCount * 100 / 3,
		RangeEnd:            maximumOperandCount * 100 / 3,
	})
}

func operandCountParameter(context *gin.Context, name string, fallback int) int {
	value, err := strconv.Atoi(context.Query(name))
	if err != nil || value < 0 || value > 3 {
		return fallback
	}
	return value
}

func toInstructionView(instruction domain.Instruction) instructionView {
	const defaultInstructionImageURL = "http://localhost:9000/riscv-assets/images/default.svg"

	imageURL := instruction.ImageURL
	if imageURL == "" {
		imageURL = defaultInstructionImageURL
	}

	return instructionView{
		ID:              instruction.ID,
		Mnemonic:        instruction.Mnemonic,
		AssemblyExample: instruction.AssemblyExample,
		Description:     instruction.Description,
		Format:          instruction.Format,
		OperandCount:    instruction.OperandCount,
		ImageURL:        imageURL,
		VideoURL:        instruction.VideoURL,
		LikeCount:       len(instruction.LikedUserIDs),
	}
}

func renderError(context *gin.Context, status int, view errorPageView) {
	context.HTML(status, "error.html", view)
}
