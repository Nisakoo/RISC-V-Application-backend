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
	server := newTestServer(t)

	tests := []struct {
		name       string
		url        string
		statusCode int
		contains   []string
		excludes   []string
	}{
		{name: "лента с корневого адреса", url: "/", statusCode: http.StatusOK, contains: []string{"<title>ADD · RISC-V</title>", "<div class=\"feed-progress\" aria-hidden=\"true\"></div>", "class=\"feed-description-toggle\">ещё", "class=\"feed-description-toggle\">меньше", "class=\"feed-like\"", "href=\"/\" aria-current=\"page\"", "<svg"}, excludes: []string{"♡", "<script"}},
		{name: "лента по идентификатору", url: "/?id=4", statusCode: http.StatusOK, contains: []string{"<title>SUB · RISC-V</title>"}},
		{name: "ссылка на следующую команду из коллекции", url: "/?id=4", statusCode: http.StatusOK, contains: []string{"<title>SUB · RISC-V</title>", "href=\"/?id=9\""}},
		{name: "циклическая ссылка после последней команды", url: "/?id=42", statusCode: http.StatusOK, contains: []string{"<title>JAL · RISC-V</title>", "href=\"/?id=1\""}},
		{name: "черновик", url: "/instruction/draft", statusCode: http.StatusOK, contains: []string{"value=\"ADD\"", "Основная навигация", "href=\"/instruction/draft\" aria-current=\"page\"", "данные формы не сохраняются"}, excludes: []string{"breadcrumbs", "<script"}},
		{name: "серверная фильтрация плитки", url: "/instructions?min_operands=2&max_operands=2", statusCode: http.StatusOK, contains: []string{"<h2 class=\"card-title\">LUI</h2>", "<h2 class=\"card-title\">JAL</h2>", ">Показать</button>", "operand-range__dot--2 active", "href=\"/instructions\" aria-current=\"page\""}, excludes: []string{"<h2 class=\"card-title\">SUB</h2>", "breadcrumbs", "operand-range__dot--0 active", "operand-range__dot--1 active", "operand-range__dot--3 active", "<script", "data-video-url"}},
		{name: "перевёрнутый диапазон нормализуется", url: "/instructions?min_operands=3&max_operands=2", statusCode: http.StatusOK, contains: []string{"name=\"min_operands\" type=\"range\" min=\"0\" max=\"3\" value=\"2\"", "name=\"max_operands\" type=\"range\" min=\"0\" max=\"3\" value=\"3\""}},
		{name: "некорректный диапазон заменяется полным", url: "/instructions?min_operands=abc&max_operands=8", statusCode: http.StatusOK, contains: []string{"name=\"min_operands\" type=\"range\" min=\"0\" max=\"3\" value=\"0\"", "name=\"max_operands\" type=\"range\" min=\"0\" max=\"3\" value=\"3\""}},
		{name: "удалённая команда исключена из плитки", url: "/instructions", statusCode: http.StatusOK, excludes: []string{"<h2 class=\"card-title\">AND</h2>"}},
		{name: "некорректный идентификатор", url: "/?id=abc", statusCode: http.StatusBadRequest, contains: []string{"<title>Некорректный идентификатор · RISC-V</title>", "Некорректный идентификатор команды.", "href=\"/\">Открыть ленту</a>"}},
		{name: "неизвестная команда", url: "/?id=999", statusCode: http.StatusNotFound, contains: []string{"<title>Команда не найдена · RISC-V</title>", "href=\"/\">Открыть ленту</a>"}},
		{name: "удалённая команда", url: "/?id=77", statusCode: http.StatusNotFound, contains: []string{"<title>Команда не найдена · RISC-V</title>"}},
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
			if navigationCount := strings.Count(response.Body.String(), `<nav class="bottom-navigation"`); navigationCount != 1 {
				t.Errorf("получено навигационных панелей: %d, ожидалась одна", navigationCount)
			}
			if activePageCount := strings.Count(response.Body.String(), `aria-current="page"`); activePageCount != 1 {
				t.Errorf("получено активных разделов навигации: %d, ожидался один", activePageCount)
			}
			if strings.Contains(response.Body.String(), "<script") {
				t.Error("страницы первой лабораторной не должны содержать JavaScript")
			}
		})
	}
}

func TestLabOneHasExactlyThreeApplicationGetControllers(t *testing.T) {
	server := newTestServer(t)
	applicationRoutes := make(map[string]string)
	for _, route := range server.router.Routes() {
		if strings.HasPrefix(route.Path, "/static/") {
			continue
		}
		applicationRoutes[route.Path] = route.Method
	}

	expectedRoutes := map[string]string{
		"/":                  http.MethodGet,
		"/instruction/draft": http.MethodGet,
		"/instructions":      http.MethodGet,
	}
	if len(applicationRoutes) != len(expectedRoutes) {
		t.Fatalf("получены прикладные маршруты %v, ожидались %v", applicationRoutes, expectedRoutes)
	}
	for path, method := range expectedRoutes {
		if applicationRoutes[path] != method {
			t.Errorf("маршрут %s использует метод %q, ожидался %q", path, applicationRoutes[path], method)
		}
	}
}

func TestStylesKeepNavigationInsideVerticalScreen(t *testing.T) {
	server := newTestServer(t)
	request := httptest.NewRequest(http.MethodGet, "/static/styles/app.css", nil)
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("получен код %d, ожидался %d", response.Code, http.StatusOK)
	}

	stylesheet := response.Body.String()
	requiredFragments := []string{
		"height: 100dvh",
		"grid-template-rows: minmax(0, 1fr) auto",
		"env(safe-area-inset-bottom, 0px)",
		"@media (prefers-reduced-motion: reduce)",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(stylesheet, fragment) {
			t.Errorf("в стилях отсутствует %q", fragment)
		}
	}
	if strings.Contains(stylesheet, "position: fixed") {
		t.Error("нижняя навигация не должна закрепляться относительно окна браузера")
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()

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
	return NewServer()
}
