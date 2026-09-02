package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLabOnePages(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	backendDirectory := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	if err := os.Chdir(backendDirectory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("не удалось восстановить рабочий каталог: %v", err)
		}
	})

	gin.SetMode(gin.TestMode)
	server := NewServer()

	tests := []struct {
		name       string
		url        string
		statusCode int
		contains   []string
		excludes   []string
	}{
		{name: "лента с корневого адреса", url: "/", statusCode: http.StatusOK, contains: []string{"<title>ADD · RISC-V</title>", "class=\"description-toggle\">ещё", "class=\"description-toggle\">меньше", "class=\"like\"", "<svg"}, excludes: []string{"♡"}},
		{name: "лента по идентификатору", url: "/?id=4", statusCode: http.StatusOK, contains: []string{"<title>SUB · RISC-V</title>"}},
		{name: "ссылка на следующую команду из коллекции", url: "/?id=4", statusCode: http.StatusOK, contains: []string{"<title>SUB · RISC-V</title>", "href=\"/?id=9\""}},
		{name: "черновик", url: "/instruction/draft", statusCode: http.StatusOK, contains: []string{"value=\"ADD\""}, excludes: []string{"breadcrumbs"}},
		{name: "серверная фильтрация плитки", url: "/instructions?min_operands=2&max_operands=2", statusCode: http.StatusOK, contains: []string{"<h2>LUI</h2>", "<h2>JAL</h2>", ">Показать</button>", "operand-range-dot-2 active"}, excludes: []string{"<h2>SUB</h2>", "breadcrumbs", "operand-range-dot-0 active", "operand-range-dot-1 active", "operand-range-dot-3 active"}},
		{name: "некорректный идентификатор", url: "/?id=abc", statusCode: http.StatusBadRequest},
		{name: "удалённая команда", url: "/?id=77", statusCode: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.url, nil)
			response := httptest.NewRecorder()
			server.router.ServeHTTP(response, request)

			if response.Code != test.statusCode {
				t.Fatalf("получен код %d, ожидался %d", response.Code, test.statusCode)
			}

			for _, fragment := range test.contains {
				if !strings.Contains(response.Body.String(), fragment) {
					t.Errorf("в ответе отсутствует %q", fragment)
				}
			}
			for _, fragment := range test.excludes {
				if strings.Contains(response.Body.String(), fragment) {
					t.Errorf("ответ не должен содержать %q", fragment)
				}
			}
		})
	}
}
