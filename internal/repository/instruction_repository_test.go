package repository

import (
	"errors"
	"reflect"
	"testing"

	"riscv-lab/internal/domain"
)

func TestInstructionRepositoryGetVisible(t *testing.T) {
	tests := []struct {
		name        string
		minimum     int
		maximum     int
		expectedIDs []int
	}{
		{
			name:        "полный диапазон без удалённой команды",
			minimum:     0,
			maximum:     3,
			expectedIDs: []int{1, 4, 9, 15, 23, 42},
		},
		{
			name:        "ровно два операнда",
			minimum:     2,
			maximum:     2,
			expectedIDs: []int{23, 42},
		},
		{
			name:        "пустой диапазон",
			minimum:     0,
			maximum:     1,
			expectedIDs: []int{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := NewInstructionRepository()
			instructions := repository.GetVisible(test.minimum, test.maximum)
			actualIDs := make([]int, 0, len(instructions))
			for _, instruction := range instructions {
				actualIDs = append(actualIDs, instruction.ID)
				if instruction.Status == domain.InstructionDeleted {
					t.Errorf("удалённая команда %d попала в доступную коллекцию", instruction.ID)
				}
			}

			if !reflect.DeepEqual(actualIDs, test.expectedIDs) {
				t.Errorf("получены идентификаторы %v, ожидались %v", actualIDs, test.expectedIDs)
			}
		})
	}
}

func TestInstructionRepositoryGetNextVisible(t *testing.T) {
	tests := []struct {
		name            string
		currentID       int
		expectedNextID  int
		expectsNotFound bool
	}{
		{
			name:           "следующая команда сохраняет порядок коллекции",
			currentID:      4,
			expectedNextID: 9,
		},
		{
			name:           "после последней команды возвращается первая",
			currentID:      42,
			expectedNextID: 1,
		},
		{
			name:            "удалённая команда отсутствует в ленте",
			currentID:       77,
			expectsNotFound: true,
		},
		{
			name:            "неизвестная команда отсутствует",
			currentID:       999,
			expectsNotFound: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := NewInstructionRepository()
			nextInstruction, err := repository.GetNextVisible(test.currentID)
			if test.expectsNotFound {
				if !errors.Is(err, ErrInstructionNotFound) {
					t.Fatalf("получена ошибка %v, ожидалась %v", err, ErrInstructionNotFound)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if nextInstruction.ID != test.expectedNextID {
				t.Errorf("получен идентификатор %d, ожидался %d", nextInstruction.ID, test.expectedNextID)
			}
		})
	}
}

func TestInstructionRepositoryGetDraft(t *testing.T) {
	repository := NewInstructionRepository()
	draft, err := repository.GetDraft()
	if err != nil {
		t.Fatal(err)
	}
	if draft.ID != 1 || draft.Status != domain.InstructionDraft {
		t.Errorf("получен черновик с ID %d и статусом %q", draft.ID, draft.Status)
	}
}
