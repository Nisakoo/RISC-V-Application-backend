package service_test

import (
	"errors"
	"testing"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/repository"
	"riscv-lab/internal/service"
)

func TestInstructionServiceFeed(t *testing.T) {
	instructionService := service.NewInstructionService(repository.NewInstructionRepository())

	firstInstruction, nextInstruction, err := instructionService.GetFeedInstruction(nil)
	if err != nil {
		t.Fatal(err)
	}
	if firstInstruction.Mnemonic != "ADD" || nextInstruction.Mnemonic != "SUB" {
		t.Fatalf("получена последовательность %s -> %s, ожидалась ADD -> SUB", firstInstruction.Mnemonic, nextInstruction.Mnemonic)
	}

	lastInstructionID := 42
	lastInstruction, firstInstructionAgain, err := instructionService.GetFeedInstruction(&lastInstructionID)
	if err != nil {
		t.Fatal(err)
	}
	if lastInstruction.Mnemonic != "JAL" || firstInstructionAgain.Mnemonic != "ADD" {
		t.Fatalf("получена последовательность %s -> %s, ожидалась JAL -> ADD", lastInstruction.Mnemonic, firstInstructionAgain.Mnemonic)
	}
}

func TestInstructionServiceRejectsDeletedInstruction(t *testing.T) {
	instructionService := service.NewInstructionService(repository.NewInstructionRepository())
	deletedInstructionID := 77

	_, _, err := instructionService.GetFeedInstruction(&deletedInstructionID)
	if !errors.Is(err, service.ErrInstructionNotFound) {
		t.Fatalf("получена ошибка %v, ожидалась %v", err, service.ErrInstructionNotFound)
	}
}

func TestInstructionServiceReturnsCanonicalDraft(t *testing.T) {
	instructionService := service.NewInstructionService(repository.NewInstructionRepository())

	draft, err := instructionService.GetDraft()
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != domain.InstructionDraft {
		t.Fatalf("получен статус %q, ожидался %q", draft.Status, domain.InstructionDraft)
	}
	if draft.AssemblyExample != "add x3, x1, x2" {
		t.Fatalf("получен пример %q, ожидался пример ADD", draft.AssemblyExample)
	}
}

func TestInstructionServiceNormalizesOperandRange(t *testing.T) {
	instructionService := service.NewInstructionService(repository.NewInstructionRepository())

	instructions, minimum, maximum := instructionService.GetVisibleByOperandRange(3, 2)
	if minimum != 2 || maximum != 3 {
		t.Fatalf("получен диапазон %d–%d, ожидался 2–3", minimum, maximum)
	}
	if len(instructions) != 6 {
		t.Fatalf("получено инструкций: %d, ожидалось 6", len(instructions))
	}
}
