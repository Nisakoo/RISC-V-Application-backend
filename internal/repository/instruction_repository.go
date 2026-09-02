package repository

import (
	"errors"

	"riscv-lab/internal/domain"
)

var ErrInstructionNotFound = errors.New("instruction not found")

type InstructionRepository struct {
	instructions []domain.Instruction
}

func NewInstructionRepository() *InstructionRepository {
	const mediaBaseURL = "http://localhost:9000/riscv-assets"

	return &InstructionRepository{instructions: []domain.Instruction{
		{
			ID:           1,
			Mnemonic:     "ADD",
			Description:  "Складывает значения двух регистров и сохраняет результат в целевой регистр.",
			Format:       "R",
			OperandCount: 3,
			ImageURL:     mediaBaseURL + "/images/ADD.png",
			VideoURL:     mediaBaseURL + "/video/ADD.mp4",
			Status:       domain.InstructionDraft,
			LikedUserIDs: userIDs(128, 0),
		},
		{
			ID:           4,
			Mnemonic:     "SUB",
			Description:  "Вычитает значение второго регистра из первого и записывает разность.",
			Format:       "R",
			OperandCount: 3,
			ImageURL:     mediaBaseURL + "/images/SUB.png",
			VideoURL:     mediaBaseURL + "/video/SUB.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: userIDs(94, 200),
		},
		{
			ID:           9,
			Mnemonic:     "ADDI",
			Description:  "Складывает значение регистра с непосредственным значением.",
			Format:       "I",
			OperandCount: 3,
			ImageURL:     mediaBaseURL + "/images/ADDI.png",
			VideoURL:     mediaBaseURL + "/video/ADDI.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: userIDs(76, 400),
		},
		{
			ID:           15,
			Mnemonic:     "BEQ",
			Description:  "Переходит по метке, если значения двух регистров равны.",
			Format:       "B",
			OperandCount: 3,
			ImageURL:     mediaBaseURL + "/images/BEQ.png",
			VideoURL:     mediaBaseURL + "/video/BEQ.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: userIDs(63, 600),
		},
		{
			ID:           23,
			Mnemonic:     "LUI",
			Description:  "Загружает 20 старших бит непосредственного значения в регистр.",
			Format:       "U",
			OperandCount: 2,
			ImageURL:     mediaBaseURL + "/images/LUI.png",
			VideoURL:     mediaBaseURL + "/video/LUI.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: userIDs(51, 800),
		},
		{
			ID:           42,
			Mnemonic:     "JAL",
			Description:  "Переходит по адресу и сохраняет адрес следующей команды в регистр.",
			Format:       "J",
			OperandCount: 2,
			ImageURL:     mediaBaseURL + "/images/JAL.png",
			VideoURL:     mediaBaseURL + "/video/JAL.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: userIDs(43, 1000),
		},
		{
			ID:           77,
			Mnemonic:     "AND",
			Description:  "Выполняет побитовое И над значениями двух регистров.",
			Format:       "R",
			OperandCount: 3,
			ImageURL:     mediaBaseURL + "/images/default.svg",
			VideoURL:     mediaBaseURL + "/video/ADD.mp4",
			Status:       domain.InstructionDeleted,
			LikedUserIDs: userIDs(12, 1200),
		},
	}}
}

func (repository *InstructionRepository) GetByID(id int) (domain.Instruction, error) {
	for _, instruction := range repository.instructions {
		if instruction.ID == id {
			return instruction, nil
		}
	}

	return domain.Instruction{}, ErrInstructionNotFound
}

func (repository *InstructionRepository) GetDraft() (domain.Instruction, error) {
	for _, instruction := range repository.instructions {
		if instruction.Status == domain.InstructionDraft {
			return instruction, nil
		}
	}

	return domain.Instruction{}, ErrInstructionNotFound
}

func (repository *InstructionRepository) GetVisible(minimumOperandCount int, maximumOperandCount int) []domain.Instruction {
	visible := make([]domain.Instruction, 0, len(repository.instructions))
	for _, instruction := range repository.instructions {
		if instruction.Status == domain.InstructionDeleted {
			continue
		}
		if instruction.OperandCount < minimumOperandCount || instruction.OperandCount > maximumOperandCount {
			continue
		}
		visible = append(visible, instruction)
	}

	return visible
}

func (repository *InstructionRepository) GetNextVisible(id int) (domain.Instruction, error) {
	visible := repository.GetVisible(0, 3)
	for index, instruction := range visible {
		if instruction.ID == id {
			return visible[(index+1)%len(visible)], nil
		}
	}

	return domain.Instruction{}, ErrInstructionNotFound
}

func userIDs(count int, offset int) []int {
	identifiers := make([]int, count)
	for index := range identifiers {
		identifiers[index] = offset + index + 1
	}
	return identifiers
}
