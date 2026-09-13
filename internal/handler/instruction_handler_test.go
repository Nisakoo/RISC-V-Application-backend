package handler

import (
	"testing"

	"riscv-lab/internal/domain"
)

func TestInstructionViewUsesDefaultImage(t *testing.T) {
	view := toInstructionView(domain.Instruction{Mnemonic: "XOR"})

	const expectedURL = "http://localhost:9000/riscv-assets/images/default.svg"
	if view.ImageURL != expectedURL {
		t.Fatalf("получен URL %q, ожидался %q", view.ImageURL, expectedURL)
	}
}
