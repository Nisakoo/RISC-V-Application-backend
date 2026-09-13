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
	server := newTestServer()

	tests := []struct {
		name       string
		url        string
		statusCode int
		contains   []string
		excludes   []string
		ordered    []string
	}{
		{
			name:       "лента без идентификатора",
			url:        "/riscv_commands/feed",
			statusCode: http.StatusOK,
			contains: []string{
				"<title>ADD · RISC-V</title>",
				"<div class=\"feed-progress\" aria-hidden=\"true\"></div>",
				"class=\"feed-description-toggle\">ещё",
				"class=\"feed-description-toggle\">меньше",
				"<dt class=\"visually-hidden\">Длина</dt><dd>32 бит</dd>",
				"class=\"feed-like\"",
				"href=\"/riscv_commands/feed\" aria-current=\"page\"",
			},
			excludes: []string{"♡", "<dt>Формат</dt>", "<dt>Операнды</dt>", "<dt>Длина</dt>", "feed-side", "<script"},
			ordered: []string{
				"<div class=\"feed-information\">",
				"<details class=\"feed-description\">",
				"<dl class=\"feed-properties\"",
				"<div class=\"feed-actions\">",
				"class=\"feed-like\"",
			},
		},
		{
			name:       "лента по идентификатору",
			url:        "/riscv_commands/feed?id=4",
			statusCode: http.StatusOK,
			contains:   []string{"<title>SUB · RISC-V</title>"},
		},
		{
			name:       "ссылка на следующую команду из коллекции",
			url:        "/riscv_commands/feed?id=4",
			statusCode: http.StatusOK,
			contains:   []string{"<title>SUB · RISC-V</title>", "href=\"/riscv_commands/feed?id=9\""},
		},
		{
			name:       "циклическая ссылка после последней команды",
			url:        "/riscv_commands/feed?id=42",
			statusCode: http.StatusOK,
			contains:   []string{"<title>JAL · RISC-V</title>", "href=\"/riscv_commands/feed?id=1\""},
		},
		{
			name:       "черновик без кнопки добавления",
			url:        "/riscv_commands/add",
			statusCode: http.StatusOK,
			contains: []string{
				"value=\"ADD\"",
				"name=\"length_bits\"",
				"value=\"32\"",
				"href=\"/riscv_commands/add\" aria-current=\"page\"",
				"данные формы не сохраняются",
			},
			excludes: []string{"Добавить в ленту", "<script"},
		},
		{
			name:       "серверная фильтрация плитки",
			url:        "/riscv_commands/grid?min_operands=2&max_operands=2",
			statusCode: http.StatusOK,
			contains: []string{
				"<h2 class=\"card-title\">LUI</h2>",
				"<h2 class=\"card-title\">JAL</h2>",
				">Показать</button>",
				"operand-range__dot--2 active",
				"href=\"/riscv_commands/grid\" aria-current=\"page\"",
				"U · 2 · 32 бит",
			},
			excludes: []string{
				"<h2 class=\"card-title\">SUB</h2>",
				"operand-range__dot--0 active",
				"operand-range__dot--1 active",
				"operand-range__dot--3 active",
				"<script",
				"data-video-url",
			},
		},
		{
			name:       "перевёрнутый диапазон нормализуется",
			url:        "/riscv_commands/grid?min_operands=3&max_operands=2",
			statusCode: http.StatusOK,
			contains: []string{
				"name=\"min_operands\" type=\"range\" min=\"0\" max=\"3\" value=\"2\"",
				"name=\"max_operands\" type=\"range\" min=\"0\" max=\"3\" value=\"3\"",
			},
		},
		{
			name:       "некорректный диапазон заменяется полным",
			url:        "/riscv_commands/grid?min_operands=abc&max_operands=8",
			statusCode: http.StatusOK,
			contains: []string{
				"name=\"min_operands\" type=\"range\" min=\"0\" max=\"3\" value=\"0\"",
				"name=\"max_operands\" type=\"range\" min=\"0\" max=\"3\" value=\"3\"",
			},
		},
		{
			name:       "удалённая команда исключена из плитки",
			url:        "/riscv_commands/grid",
			statusCode: http.StatusOK,
			excludes:   []string{"<h2 class=\"card-title\">AND</h2>"},
		},
		{
			name:       "некорректный идентификатор",
			url:        "/riscv_commands/feed?id=abc",
			statusCode: http.StatusBadRequest,
			contains: []string{
				"<title>Некорректный идентификатор · RISC-V</title>",
				"Некорректный идентификатор команды.",
				"href=\"/riscv_commands/feed\">Открыть ленту</a>",
			},
		},
		{
			name:       "неизвестная команда",
			url:        "/riscv_commands/feed?id=999",
			statusCode: http.StatusNotFound,
			contains: []string{
				"<title>Команда не найдена · RISC-V</title>",
				"href=\"/riscv_commands/feed\">Открыть ленту</a>",
			},
		},
		{
			name:       "удалённая команда",
			url:        "/riscv_commands/feed?id=77",
			statusCode: http.StatusNotFound,
			contains:   []string{"<title>Команда не найдена · RISC-V</title>"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.url, nil)
			response := httptest.NewRecorder()
			server.router.ServeHTTP(response, request)

			if response.Code != test.statusCode {
				t.Fatalf("получен код %d, ожидался %d", response.Code, test.statusCode)
			}

			body := response.Body.String()
			for _, fragment := range test.contains {
				if !strings.Contains(body, fragment) {
					t.Errorf("в ответе отсутствует %q", fragment)
				}
			}
			for _, fragment := range test.excludes {
				if strings.Contains(body, fragment) {
					t.Errorf("ответ не должен содержать %q", fragment)
				}
			}
			assertFragmentsOrder(t, body, test.ordered)
			if navigationCount := strings.Count(body, `<nav class="bottom-navigation"`); navigationCount != 1 {
				t.Errorf("получено навигационных панелей: %d, ожидалась одна", navigationCount)
			}
			if activePageCount := strings.Count(body, `aria-current="page"`); activePageCount != 1 {
				t.Errorf("получено активных разделов навигации: %d, ожидался один", activePageCount)
			}
			if strings.Contains(body, "<script") {
				t.Error("страницы первой лабораторной не должны содержать JavaScript")
			}
		})
	}
}

func TestLabOneHasExactlyThreeRiscVCommandGetControllers(t *testing.T) {
	server := newTestServer()
	applicationRoutes := make(map[string]string)
	for _, registeredRoute := range server.router.Routes() {
		if !strings.HasPrefix(registeredRoute.Path, "/riscv_commands/") {
			continue
		}
		applicationRoutes[registeredRoute.Path] = registeredRoute.Method
	}

	expectedRoutes := map[string]string{
		"/riscv_commands/feed": http.MethodGet,
		"/riscv_commands/add":  http.MethodGet,
		"/riscv_commands/grid": http.MethodGet,
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

func TestRootRedirectsToFeed(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)

	if response.Code != http.StatusFound {
		t.Fatalf("корневой маршрут вернул код %d, ожидался %d", response.Code, http.StatusFound)
	}
	if location := response.Header().Get("Location"); location != "/riscv_commands/feed" {
		t.Errorf("корневой маршрут перенаправляет на %q, ожидался %q", location, "/riscv_commands/feed")
	}
}

func TestOldApplicationRoutesAreNotServed(t *testing.T) {
	server := newTestServer()
	oldRoutes := []string{"/instruction/draft", "/instructions"}
	for _, oldRoute := range oldRoutes {
		request := httptest.NewRequest(http.MethodGet, oldRoute, nil)
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf("старый маршрут %s вернул код %d, ожидался %d", oldRoute, response.Code, http.StatusNotFound)
		}
	}
}

func TestStylesKeepNavigationInsideVerticalScreen(t *testing.T) {
	server := newTestServer()
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

func TestStylesAlignOperandRangeThumbsWithTrack(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/static/styles/app.css", nil)
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("получен код %d, ожидался %d", response.Code, http.StatusOK)
	}

	stylesheet := response.Body.String()
	requiredFragments := []string{
		".operand-range .form-range::-webkit-slider-thumb {",
		"margin-top: 0",
		".operand-range .form-range::-moz-range-thumb {",
		".operand-range__dot {",
		"top: 10px",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(stylesheet, fragment) {
			t.Errorf("в стилях ползунка отсутствует %q", fragment)
		}
	}
	if strings.Contains(stylesheet, "margin-top: -9px") {
		t.Error("бегунки фильтра не должны смещаться выше дорожки")
	}
}

func assertFragmentsOrder(t *testing.T, body string, fragments []string) {
	t.Helper()
	previousIndex := -1
	for _, fragment := range fragments {
		index := strings.Index(body, fragment)
		if index == -1 {
			t.Errorf("в ответе отсутствует %q", fragment)
			return
		}
		if index <= previousIndex {
			t.Errorf("фрагмент %q расположен в неверном порядке", fragment)
			return
		}
		previousIndex = index
	}
}

func TestMain(testingMain *testing.M) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	backendDirectory := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	if err := os.Chdir(backendDirectory); err != nil {
		panic(err)
	}

	gin.SetMode(gin.TestMode)
	exitCode := testingMain.Run()
	if err := os.Chdir(workingDirectory); err != nil {
		panic(err)
	}
	os.Exit(exitCode)
}

func newTestServer() *Server {
	return NewServer()
}
