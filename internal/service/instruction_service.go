package service

import (
	"errors"

	"riscv-lab/internal/domain"
)

var ErrInstructionNotFound = errors.New("instruction not found")

type InstructionRepository interface {
	GetByID(id int) (domain.Instruction, error)
	GetDraft() (domain.Instruction, error)
	GetVisible(minimumOperandCount int, maximumOperandCount int) []domain.Instruction
	// GetNextVisible сохраняет порядок ленты независимо от способа хранения команд.
	GetNextVisible(id int) (domain.Instruction, error)
}

type InstructionService struct {
	repository InstructionRepository
}

func NewInstructionService(repository InstructionRepository) *InstructionService {
	return &InstructionService{repository: repository}
}

func (service *InstructionService) GetFeedInstruction(id *int) (domain.Instruction, domain.Instruction, error) {
	var instruction domain.Instruction
	var err error

	if id == nil {
		visibleInstructions := service.repository.GetVisible(0, 3)
		if len(visibleInstructions) == 0 {
			return domain.Instruction{}, domain.Instruction{}, ErrInstructionNotFound
		}
		instruction = visibleInstructions[0]
	} else {
		instruction, err = service.repository.GetByID(*id)
		if err != nil || instruction.Status == domain.InstructionDeleted {
			return domain.Instruction{}, domain.Instruction{}, ErrInstructionNotFound
		}
	}

	nextInstruction, err := service.repository.GetNextVisible(instruction.ID)
	if err != nil {
		return domain.Instruction{}, domain.Instruction{}, ErrInstructionNotFound
	}

	return instruction, nextInstruction, nil
}

func (service *InstructionService) GetDraft() (domain.Instruction, error) {
	instruction, err := service.repository.GetDraft()
	if err != nil {
		return domain.Instruction{}, ErrInstructionNotFound
	}
	return instruction, nil
}

func (service *InstructionService) GetVisibleByOperandRange(minimumOperandCount int, maximumOperandCount int) ([]domain.Instruction, int, int) {
	if minimumOperandCount > maximumOperandCount {
		minimumOperandCount, maximumOperandCount = maximumOperandCount, minimumOperandCount
	}

	return service.repository.GetVisible(minimumOperandCount, maximumOperandCount), minimumOperandCount, maximumOperandCount
}
