package app

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/handler"
	"riscv-lab/internal/repository"
	"riscv-lab/internal/service"
)

type applicationRepository struct {
	commands []domain.RiscVCommand
	draft    *domain.RiscVCommand
}

func (fake *applicationRepository) GetByID(_ context.Context, id int64) (domain.RiscVCommand, error) {
	for _, command := range fake.commands {
		if command.ID == id {
			return command, nil
		}
	}
	if fake.draft != nil && fake.draft.ID == id {
		return *fake.draft, nil
	}
	return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
}

func (fake *applicationRepository) GetDraftByCreatorID(_ context.Context, _ int64) (domain.RiscVCommand, error) {
	if fake.draft == nil {
		return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
	}
	return *fake.draft, nil
}

func (fake *applicationRepository) GetVisible(_ context.Context, minimum int, maximum int) ([]domain.RiscVCommand, error) {
	result := make([]domain.RiscVCommand, 0)
	for _, command := range fake.commands {
		if command.Status == domain.RiscVCommandPublished && command.OperandCount >= minimum && command.OperandCount <= maximum {
			result = append(result, command)
		}
	}
	return result, nil
}

func (fake *applicationRepository) GetNextVisible(_ context.Context, id int64) (domain.RiscVCommand, error) {
	visible, _ := fake.GetVisible(context.Background(), 0, 3)
	for index, command := range visible {
		if command.ID == id {
			return visible[(index+1)%len(visible)], nil
		}
	}
	return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
}

func (fake *applicationRepository) CreateDraft(_ context.Context, creatorID int64, input domain.CreateRiscVCommandDraft, imageKey string, videoKey string) (domain.RiscVCommand, error) {
	if fake.draft != nil {
		return domain.RiscVCommand{}, repository.ErrDraftAlreadyExists
	}
	command := domain.RiscVCommand{ID: 100, Mnemonic: input.Mnemonic, ImageKey: imageKey, VideoKey: videoKey, Status: domain.RiscVCommandDraft, CreatorID: creatorID, LengthBits: 32}
	fake.draft = &command
	return command, nil
}

func (fake *applicationRepository) PublishDraft(_ context.Context, id int64, _ int64, input domain.PublishRiscVCommandDraft) (domain.RiscVCommand, error) {
	if fake.draft == nil || fake.draft.ID != id {
		return domain.RiscVCommand{}, repository.ErrRiscVCommandNotFound
	}
	command := *fake.draft
	command.AssemblyExample = input.AssemblyExample
	command.Description = input.Description
	command.Format = input.Format
	command.OperandCount = input.OperandCount
	command.LengthBits = input.LengthBits
	command.Status = domain.RiscVCommandPublished
	fake.commands = append(fake.commands, command)
	fake.draft = nil
	return command, nil
}

func (fake *applicationRepository) SoftDelete(_ context.Context, id int64) error {
	for index := range fake.commands {
		if fake.commands[index].ID == id && fake.commands[index].Status != domain.RiscVCommandDeleted {
			fake.commands[index].Status = domain.RiscVCommandDeleted
			return nil
		}
	}
	return repository.ErrRiscVCommandNotFound
}

type applicationMedia struct{}

func (applicationMedia) Put(_ context.Context, directory string, extension string, _ string, reader io.Reader, _ int64) (string, error) {
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return "", err
	}
	return directory + "/uploaded" + extension, nil
}

func (applicationMedia) Remove(_ context.Context, _ string) error { return nil }

func (applicationMedia) URL(key string) string {
	if key == "" {
		return "http://localhost:9000/riscv-assets/images/default.svg"
	}
	return "http://localhost:9000/riscv-assets/" + key
}

func TestLabTwoHasExactlySixRiscVCommandMethods(t *testing.T) {
	server, _ := newTestServer()
	routes := make([]string, 0)
	for _, registeredRoute := range server.router.Routes() {
		if strings.HasPrefix(registeredRoute.Path, "/riscv_commands/") {
			routes = append(routes, registeredRoute.Method+" "+registeredRoute.Path)
		}
	}
	sort.Strings(routes)
	expected := []string{
		"GET /riscv_commands/add",
		"GET /riscv_commands/feed",
		"GET /riscv_commands/grid",
		"POST /riscv_commands/add",
		"POST /riscv_commands/delete",
		"POST /riscv_commands/publish",
	}
	if !reflect.DeepEqual(routes, expected) {
		t.Fatalf("получены маршруты %v, ожидались %v", routes, expected)
	}
}

func TestLabTwoPagesReadRiscVCommands(t *testing.T) {
	server, _ := newTestServer()
	tests := []struct {
		url      string
		contains []string
	}{
		{url: "/riscv_commands/feed", contains: []string{"<title>ADD · RISC-V</title>", "href=\"/riscv_commands/feed?id=4\"", ">4</small>"}},
		{url: "/riscv_commands/feed?id=4", contains: []string{"<title>SUB · RISC-V</title>", "href=\"/riscv_commands/feed?id=23\""}},
		{url: "/riscv_commands/grid?min_operands=2&max_operands=2", contains: []string{"<h2 class=\"card-title\">LUI</h2>", "action=\"/riscv_commands/delete\""}},
		{url: "/riscv_commands/add", contains: []string{"Создайте черновик команды RISC-V", "enctype=\"multipart/form-data\"", ">Далее</button>"}},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, test.url, nil)
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Errorf("%s вернул %d, ожидался 200", test.url, response.Code)
		}
		for _, fragment := range test.contains {
			if !strings.Contains(response.Body.String(), fragment) {
				t.Errorf("в ответе %s отсутствует %q", test.url, fragment)
			}
		}
		if strings.Contains(response.Body.String(), "<script") {
			t.Errorf("страница %s содержит JavaScript", test.url)
		}
	}
}

func TestLabTwoCreateAndPublishFlow(t *testing.T) {
	server, fakeRepository := newTestServer()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("mnemonic", "xor")
	writeMultipartFile(t, writer, "riscv_command_image", "xor.png", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"))
	writeMultipartFile(t, writer, "riscv_command_video", "xor.mp4", []byte("\x00\x00\x00\x14ftypmp42\x00\x00\x00\x00mp42"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/riscv_commands/add", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || fakeRepository.draft == nil || fakeRepository.draft.Mnemonic != "XOR" {
		t.Fatalf("черновик не создан: код %d, черновик %+v", response.Code, fakeRepository.draft)
	}

	publishForm := url.Values{
		"assembly_example": {"xor x3, x1, x2"},
		"description":      {"Выполняет побитовое исключающее ИЛИ для двух регистров."},
		"format":           {"R"},
		"operand_count":    {"3"},
		"length_bits":      {"32"},
	}
	request = httptest.NewRequest(http.MethodPost, "/riscv_commands/publish", strings.NewReader(publishForm.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response = httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || fakeRepository.draft != nil {
		t.Fatalf("черновик не опубликован: код %d, тело %s", response.Code, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/riscv_commands/feed?id=100" {
		t.Fatalf("получено перенаправление %q", location)
	}
}

func TestLabTwoDeleteHidesCommandAndDirectURL(t *testing.T) {
	server, _ := newTestServer()
	request := httptest.NewRequest(http.MethodPost, "/riscv_commands/delete", strings.NewReader("riscv_command_id=4"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("удаление вернуло %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/riscv_commands/feed?id=4", nil)
	response = httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("удалённая команда вернула %d, ожидался 404", response.Code)
	}
}

func TestRootRedirectsToRiscVCommandFeed(t *testing.T) {
	server, _ := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusFound || response.Header().Get("Location") != "/riscv_commands/feed" {
		t.Fatalf("корневой URL вернул %d и Location %q", response.Code, response.Header().Get("Location"))
	}
}

func writeMultipartFile(t *testing.T, writer *multipart.Writer, field string, name string, contents []byte) {
	t.Helper()
	part, err := writer.CreateFormFile(field, name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatal(err)
	}
}

func newTestServer() (*Server, *applicationRepository) {
	commands := []domain.RiscVCommand{
		{ID: 1, Mnemonic: "ADD", Description: "Складывает значения исходных регистров.", Format: domain.InstructionFormatR, OperandCount: 3, LengthBits: 32, ImageKey: "images/ADD.png", VideoKey: "video/ADD.mp4", Status: domain.RiscVCommandPublished, LikeCount: 4},
		{ID: 4, Mnemonic: "SUB", Description: "Вычитает значение второго регистра.", Format: domain.InstructionFormatR, OperandCount: 3, LengthBits: 32, ImageKey: "images/SUB.png", VideoKey: "video/SUB.mp4", Status: domain.RiscVCommandPublished, LikeCount: 3},
		{ID: 23, Mnemonic: "LUI", Description: "Загружает верхнее immediate.", Format: domain.InstructionFormatU, OperandCount: 2, LengthBits: 32, ImageKey: "images/LUI.png", VideoKey: "video/LUI.mp4", Status: domain.RiscVCommandPublished, LikeCount: 2},
	}
	fakeRepository := &applicationRepository{commands: commands}
	media := applicationMedia{}
	riscvCommandService := service.NewRiscVCommandService(fakeRepository, media)
	riscvCommandHandler := handler.NewRiscVCommandHandler(riscvCommandService, media)
	return NewServer(":0", riscvCommandHandler), fakeRepository
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
