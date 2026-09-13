package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
)

type testMediaURLProvider struct{}

func (testMediaURLProvider) URL(key string) string {
	if key == "" {
		return "http://localhost:9000/riscv-assets/images/default.svg"
	}
	return "http://localhost:9000/riscv-assets/" + key
}

func TestRiscVCommandViewUsesDatabaseMediaKeys(t *testing.T) {
	handler := &RiscVCommandHandler{media: testMediaURLProvider{}}
	view := handler.toRiscVCommandView(domain.RiscVCommand{
		ID:              9,
		Mnemonic:        "ADDI",
		AssemblyExample: "addi x2, x1, 3",
		Description:     "Сложение с immediate.",
		Format:          domain.InstructionFormatI,
		OperandCount:    3,
		LengthBits:      32,
		ImageKey:        "images/ADDI.png",
		VideoKey:        "video/ADDI.mp4",
		LikeCount:       3,
	})
	if view.ImageURL != "http://localhost:9000/riscv-assets/images/ADDI.png" || view.VideoURL != "http://localhost:9000/riscv-assets/video/ADDI.mp4" {
		t.Fatalf("медиа-URL сформированы неверно: %+v", view)
	}
	if view.AssemblyExample != "addi x2, x1, 3" || view.LikeCount != 3 {
		t.Fatalf("предметные поля преобразованы неверно: %+v", view)
	}
}

func TestOperandCountParameter(t *testing.T) {
	tests := []struct {
		query    string
		fallback int
		expected int
	}{
		{query: "0", fallback: 2, expected: 0},
		{query: "3", fallback: 1, expected: 3},
		{query: "", fallback: 2, expected: 2},
		{query: "R", fallback: 1, expected: 1},
		{query: "-1", fallback: 0, expected: 0},
		{query: "4", fallback: 3, expected: 3},
	}

	gin.SetMode(gin.TestMode)
	for _, test := range tests {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest(http.MethodGet, "/?operand_count="+test.query, nil)
		if actual := operandCountParameter(context, "operand_count", test.fallback); actual != test.expected {
			t.Errorf("для %q получено %d, ожидалось %d", test.query, actual, test.expected)
		}
	}
}

func TestPublicationInputPreservesAssemblyExample(t *testing.T) {
	form := url.Values{
		"assembly_example": {"sub x3, x1, x2"},
		"description":      {"Вычитает значение второго исходного регистра."},
		"format":           {"R"},
		"operand_count":    {"3"},
		"length_bits":      {"32"},
	}
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	context.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	input, err := publicationInput(context)
	if err != nil {
		t.Fatal(err)
	}
	if input.AssemblyExample != "sub x3, x1, x2" || input.Format != domain.InstructionFormatR {
		t.Fatalf("получены неверные данные публикации: %+v", input)
	}
}
