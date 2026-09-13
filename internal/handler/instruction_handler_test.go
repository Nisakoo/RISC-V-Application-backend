package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
)

func TestInstructionViewUsesDefaultImage(t *testing.T) {
	view := toInstructionView(domain.Instruction{Mnemonic: "XOR"})

	const expectedURL = "http://localhost:9000/riscv-assets/images/default.svg"
	if view.ImageURL != expectedURL {
		t.Fatalf("получен URL %q, ожидался %q", view.ImageURL, expectedURL)
	}
}

func TestOperandCountParameter(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		fallback int
		expected int
	}{
		{name: "минимальная граница", query: "0", fallback: 2, expected: 0},
		{name: "максимальная граница", query: "3", fallback: 1, expected: 3},
		{name: "параметр отсутствует", fallback: 2, expected: 2},
		{name: "нечисловое значение", query: "R", fallback: 1, expected: 1},
		{name: "значение ниже диапазона", query: "-1", fallback: 0, expected: 0},
		{name: "значение выше диапазона", query: "4", fallback: 3, expected: 3},
	}

	gin.SetMode(gin.TestMode)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(http.MethodGet, "/?operand_count="+test.query, nil)

			actual := operandCountParameter(context, "operand_count", test.fallback)
			if actual != test.expected {
				t.Errorf("получено значение %d, ожидалось %d", actual, test.expected)
			}
		})
	}
}

func TestInstructionViewMapsDisplayFields(t *testing.T) {
	instruction := domain.Instruction{
		ID:              9,
		Mnemonic:        "ADDI",
		AssemblyExample: "addi x2, x1, 3",
		Description:     "Сложение с immediate.",
		Format:          domain.InstructionFormatI,
		OperandCount:    3,
		LengthBits:      32,
		ImageURL:        "/images/ADDI.png",
		VideoURL:        "/video/ADDI.mp4",
		LikedUserIDs:    []int{11, 12, 13},
	}

	view := toInstructionView(instruction)
	if view.ID != instruction.ID || view.Mnemonic != instruction.Mnemonic {
		t.Errorf("основные поля команды преобразованы неверно: %+v", view)
	}
	if view.Format != instruction.Format || view.OperandCount != 3 || view.LengthBits != 32 {
		t.Errorf("параметры команды преобразованы неверно: %+v", view)
	}
	if view.ImageURL != instruction.ImageURL || view.VideoURL != instruction.VideoURL {
		t.Errorf("медиа команды преобразованы неверно: %+v", view)
	}
	if view.LikeCount != 3 {
		t.Errorf("получено лайков %d, ожидалось 3", view.LikeCount)
	}
}
