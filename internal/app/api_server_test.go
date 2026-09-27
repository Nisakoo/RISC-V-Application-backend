package app

import (
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/handler"
	"riscv-lab/internal/repository"
	"riscv-lab/internal/service"
)

type apiApplicationRepository struct {
	*applicationRepository
}

func (fake *apiApplicationRepository) SoftDeleteOwned(_ context.Context, id, creatorID int64) error {
	for index := range fake.commands {
		if fake.commands[index].ID == id && fake.commands[index].CreatorID == creatorID && fake.commands[index].Status != domain.RiscVCommandDeleted {
			fake.commands[index].Status = domain.RiscVCommandDeleted
			return nil
		}
	}
	return repository.ErrRiscVCommandNotFound
}

func (fake *apiApplicationRepository) SetLike(_ context.Context, id, _ int64, liked bool) (int64, error) {
	for index := range fake.commands {
		if fake.commands[index].ID == id && fake.commands[index].Status == domain.RiscVCommandPublished {
			if liked {
				fake.commands[index].LikeCount = 1
			} else {
				fake.commands[index].LikeCount = 0
			}
			return fake.commands[index].LikeCount, nil
		}
	}
	return 0, repository.ErrRiscVCommandNotFound
}

func (fake *apiApplicationRepository) RegisterUser(_ context.Context, _, _ string) (int64, error) {
	return 10, nil
}

type apiApplicationMedia struct{ applicationMedia }

func (apiApplicationMedia) Upload(_ context.Context, kind string, _ *multipart.FileHeader) (string, error) {
	return kind + "/test.png", nil
}

func (apiApplicationMedia) Remove(_ context.Context, _ string) error { return nil }

func newAPITestServer() *Server {
	fake := &apiApplicationRepository{applicationRepository: &applicationRepository{commands: []domain.RiscVCommand{
		{ID: 1, Mnemonic: "ADD", Status: domain.RiscVCommandPublished, CreatorID: 1, OperandCount: 3},
		{ID: 2, Mnemonic: "SUB", Status: domain.RiscVCommandPublished, CreatorID: 2, OperandCount: 3},
		{ID: 3, Mnemonic: "OR", Status: domain.RiscVCommandDeleted, CreatorID: 1, OperandCount: 3},
	}}}
	media := apiApplicationMedia{}
	return NewServer(":0", handler.NewRiscVCommandHandler(service.NewRiscVCommandService(fake), media), handler.NewRiscVAPIHandler(service.NewRiscVAPIService(fake), media))
}

func TestLabThreeHasTenAPIMethods(t *testing.T) {
	server := newAPITestServer()
	actual := make([]string, 0)
	for _, route := range server.router.Routes() {
		if strings.HasPrefix(route.Path, "/api/") {
			actual = append(actual, route.Method+" "+route.Path)
		}
	}
	sort.Strings(actual)
	expected := []string{
		"DELETE /api/riscv_commands/:id",
		"GET /api/riscv_commands",
		"GET /api/riscv_commands/draft",
		"GET /api/riscv_commands/feed",
		"POST /api/riscv_commands",
		"POST /api/riscv_commands/:id/like",
		"POST /api/users",
		"POST /api/users/authentication",
		"POST /api/users/logout",
		"PUT /api/riscv_commands/:id/publish",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("маршруты API: %v", actual)
	}
}

func TestLabThreeAPIHidesDeletedAndValidatesInput(t *testing.T) {
	server := newAPITestServer()
	cases := []struct {
		method, path, body string
		status             int
		contains, absent   string
	}{
		{http.MethodGet, "/api/riscv_commands", "", 200, `"mnemonic":"ADD"`, `"mnemonic":"OR"`},
		{http.MethodGet, "/api/riscv_commands/feed?id=3", "", 404, `"status":"fail"`, ""},
		{http.MethodGet, "/api/riscv_commands/feed?id=-1", "", 400, `"status":"fail"`, ""},
		{http.MethodPost, "/api/riscv_commands/1/like", `{"like":2}`, 400, `"status":"fail"`, ""},
		{http.MethodPut, "/api/riscv_commands/1/publish", `{"description":"Достаточное описание","operand_count":3,"length_bits":32,"creator_id":2}`, 400, `"status":"fail"`, ""},
		{http.MethodDelete, "/api/riscv_commands/2", "", 404, `"status":"fail"`, ""},
	}
	for _, test := range cases {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		if response.Code != test.status || !strings.Contains(response.Body.String(), test.contains) || (test.absent != "" && strings.Contains(response.Body.String(), test.absent)) {
			t.Errorf("%s %s: код %d, тело %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}
