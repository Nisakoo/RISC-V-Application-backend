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
		{name: "лента с корневого адреса", url: "/", statusCode: http.StatusOK, contains: []string{"<title>ADD · RISC-V</title>"}},
		{name: "лента по идентификатору", url: "/?id=4", statusCode: http.StatusOK, contains: []string{"<title>SUB · RISC-V</title>"}},
		{name: "следующая команда из коллекции", url: "/?id=4&next=true", statusCode: http.StatusOK, contains: []string{"<title>ADDI · RISC-V</title>"}},
		{name: "черновик", url: "/instruction/draft", statusCode: http.StatusOK, contains: []string{"value=\"ADD\""}},
		{name: "серверная фильтрация плитки", url: "/instructions?operands=2", statusCode: http.StatusOK, contains: []string{"<h2>LUI</h2>", "<h2>JAL</h2>"}, excludes: []string{"<h2>SUB</h2>"}},
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
