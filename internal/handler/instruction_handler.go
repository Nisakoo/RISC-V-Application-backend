package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
)

type InstructionHandler struct {
	repository *repository.InstructionRepository
}

type instructionView struct {
	ID           int
	Mnemonic     string
	Description  string
	Format       string
	OperandCount int
	ImageURL     string
	VideoURL     string
	LikeCount    int
}

func NewInstructionHandler(repository *repository.InstructionRepository) *InstructionHandler {
	return &InstructionHandler{repository: repository}
}

func (handler *InstructionHandler) Feed(context *gin.Context) {
	idParameter := context.Query("id")
	var instruction domain.Instruction
	var err error

	if idParameter == "" {
		visibleInstructions := handler.repository.GetVisible(0, 3)
		if len(visibleInstructions) == 0 {
			context.String(http.StatusNotFound, "Нет доступных команд")
			return
		}
		instruction = visibleInstructions[0]
	} else {
		id, conversionError := strconv.Atoi(idParameter)
		if conversionError != nil {
			context.String(http.StatusBadRequest, "Некорректный идентификатор команды")
			return
		}

		instruction, err = handler.repository.GetByID(id)
		if err != nil || instruction.Status == domain.InstructionDeleted {
			context.String(http.StatusNotFound, "Команда не найдена")
			return
		}

	}

	if err != nil {
		context.String(http.StatusNotFound, "Следующая команда не найдена")
		return
	}

	nextInstruction, err := handler.repository.GetNextVisible(instruction.ID)
	if err != nil {
		context.String(http.StatusNotFound, "Следующая команда не найдена")
		return
	}

	context.HTML(http.StatusOK, "feed.html", gin.H{
		"Instruction":       toInstructionView(instruction),
		"NextInstructionID": nextInstruction.ID,
		"ActivePage":        "feed",
	})
}

func (handler *InstructionHandler) Draft(context *gin.Context) {
	instruction, err := handler.repository.GetDraft()
	if err != nil {
		context.String(http.StatusNotFound, "Черновик команды не найден")
		return
	}

	context.HTML(http.StatusOK, "draft.html", gin.H{
		"Instruction": toInstructionView(instruction),
		"ActivePage":  "draft",
	})
}

func (handler *InstructionHandler) Grid(context *gin.Context) {
	minimumOperandCount := operandCountParameter(context, "min_operands", 0)
	maximumOperandCount := operandCountParameter(context, "max_operands", 3)
	if minimumOperandCount > maximumOperandCount {
		minimumOperandCount, maximumOperandCount = maximumOperandCount, minimumOperandCount
	}

	instructions := handler.repository.GetVisible(minimumOperandCount, maximumOperandCount)
	views := make([]instructionView, 0, len(instructions))
	for _, instruction := range instructions {
		views = append(views, toInstructionView(instruction))
	}

	context.HTML(http.StatusOK, "grid.html", gin.H{
		"Instructions":        views,
		"MinimumOperandCount": minimumOperandCount,
		"MaximumOperandCount": maximumOperandCount,
		"RangeStart":          minimumOperandCount * 100 / 3,
		"RangeEnd":            maximumOperandCount * 100 / 3,
		"ActivePage":          "grid",
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
	return instructionView{
		ID:           instruction.ID,
		Mnemonic:     instruction.Mnemonic,
		Description:  instruction.Description,
		Format:       instruction.Format,
		OperandCount: instruction.OperandCount,
		ImageURL:     instruction.ImageURL,
		VideoURL:     instruction.VideoURL,
		LikeCount:    len(instruction.LikedUserIDs),
	}
}
