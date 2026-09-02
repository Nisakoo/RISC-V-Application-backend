package repository

import (
	"errors"
	"sort"

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
			LikedUserIDs: []int{3, 8, 11, 14, 18, 21, 27, 34, 39, 42, 47, 53, 61, 66, 72, 80, 84, 91, 96, 102, 109, 113, 119, 125, 130, 138, 144, 151, 157, 163, 169, 174, 181, 186, 192, 199, 204, 211, 217, 223, 228, 235, 241, 247, 252, 259, 265, 271, 276, 283, 289, 294, 301, 307, 312, 319, 325, 331, 336, 343, 349, 355, 360, 367, 373, 379, 384, 391, 397, 403, 408, 415, 421, 427, 432, 439, 445, 451, 456, 463, 469, 475, 480, 487, 493, 499, 504, 511, 517, 523, 528, 535, 541, 547, 552, 559, 565, 571, 576, 583, 589, 595, 600, 607, 613, 619, 624, 631, 637, 643, 648, 655, 661, 667, 672, 679, 685, 691, 696, 703, 709, 715, 720, 727, 733, 739, 744, 751, 757},
		},
		{
			ID:           2,
			Mnemonic:     "SUB",
			Description:  "Вычитает значение второго регистра из первого и записывает разность.",
			Format:       "R",
			OperandCount: 3,
			ImageURL:     mediaBaseURL + "/images/SUB.png",
			VideoURL:     mediaBaseURL + "/video/SUB.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94},
		},
		{
			ID:           3,
			Mnemonic:     "ADDI",
			Description:  "Складывает значение регистра с непосредственным значением.",
			Format:       "I",
			OperandCount: 3,
			ImageURL:     mediaBaseURL + "/images/ADDI.png",
			VideoURL:     mediaBaseURL + "/video/ADDI.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76},
		},
		{
			ID:           4,
			Mnemonic:     "BEQ",
			Description:  "Переходит по метке, если значения двух регистров равны.",
			Format:       "B",
			OperandCount: 3,
			ImageURL:     mediaBaseURL + "/images/BEQ.png",
			VideoURL:     mediaBaseURL + "/video/BEQ.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63},
		},
		{
			ID:           5,
			Mnemonic:     "LUI",
			Description:  "Загружает 20 старших бит непосредственного значения в регистр.",
			Format:       "U",
			OperandCount: 2,
			ImageURL:     mediaBaseURL + "/images/LUI.png",
			VideoURL:     mediaBaseURL + "/video/LUI.mp4",
			Status:       domain.InstructionPublished,
			LikedUserIDs: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51},
		},
		{
			ID:           6,
			Mnemonic:     "JAL",
			Description:  "Переходит по адресу и сохраняет адрес следующей команды в регистр.",
			Format:       "J",
			OperandCount: 2,
			ImageURL:     mediaBaseURL + "/images/JAL.png",
			VideoURL:     mediaBaseURL + "/video/JAL.mp4",
			Status:       domain.InstructionDeleted,
			LikedUserIDs: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43},
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

func (repository *InstructionRepository) GetVisible(operandCount *int) []domain.Instruction {
	visible := make([]domain.Instruction, 0, len(repository.instructions))
	for _, instruction := range repository.instructions {
		if instruction.Status == domain.InstructionDeleted {
			continue
		}
		if operandCount != nil && instruction.OperandCount != *operandCount {
			continue
		}
		visible = append(visible, instruction)
	}

	sort.Slice(visible, func(left, right int) bool {
		return visible[left].ID < visible[right].ID
	})

	return visible
}

func (repository *InstructionRepository) GetNextVisible(id int) (domain.Instruction, error) {
	visible := repository.GetVisible(nil)
	for index, instruction := range visible {
		if instruction.ID == id {
			return visible[(index+1)%len(visible)], nil
		}
	}

	return domain.Instruction{}, ErrInstructionNotFound
}
