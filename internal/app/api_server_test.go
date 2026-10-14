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
	liked map[int64]bool
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
			if fake.liked == nil {
				fake.liked = map[int64]bool{}
			}
			fake.liked[id] = liked
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

func (fake *apiApplicationRepository) IsLiked(_ context.Context, id, _ int64) (bool, error) {
	return fake.liked[id], nil
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
		{http.MethodGet, "/api/riscv_commands/feed?id=3", "", 404, `"message":`, `"status"`},
		{http.MethodGet, "/api/riscv_commands/feed?id=-1", "", 400, `"message":`, `"status"`},
		{http.MethodGet, "/api/riscv_commands/feed?next=true", "", 400, `"message":`, `"status"`},
		{http.MethodPost, "/api/riscv_commands/1/like", `{"like":2}`, 400, `"message":`, `"status"`},
		{http.MethodPut, "/api/riscv_commands/1/publish", `{"description":"Достаточное описание","operand_count":3,"length_bits":32,"creator_id":2}`, 400, `"message":`, `"status"`},
		{http.MethodDelete, "/api/riscv_commands/2", "", 404, `"message":`, `"status"`},
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

func TestLabThreeAPIReturnsDataWithoutStatusWrapper(t *testing.T) {
	server := newAPITestServer()
	cases := []struct {
		method, path, body string
		status             int
		prefix, contains   string
	}{
		{http.MethodGet, "/api/riscv_commands", "", 200, `[{`, `"is_creator":1`},
		{http.MethodGet, "/api/riscv_commands/feed?id=1", "", 200, `{"id":1`, `"is_liked":0,"next_id":2`},
		{http.MethodPost, "/api/riscv_commands/1/like", `{"like":1}`, 200, `{"like":1`, `"like_count":1`},
		{http.MethodGet, "/api/riscv_commands/feed?id=1", "", 200, `{"id":1`, `"is_liked":1,"next_id":2`},
		{http.MethodGet, "/api/riscv_commands/feed?id=1&next=true", "", 200, `{"id":2`, `"is_liked":0`},
		{http.MethodDelete, "/api/riscv_commands/1", "", 204, ``, ``},
	}
	for _, test := range cases {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		body := response.Body.String()
		if response.Code != test.status || !strings.HasPrefix(body, test.prefix) || !strings.Contains(body, test.contains) || strings.Contains(body, `"data"`) || strings.Contains(body, `"success"`) || strings.Contains(body, `"creator":{`) {
			t.Errorf("%s %s: код %d, тело %s", test.method, test.path, response.Code, body)
		}
		if test.status == http.StatusNoContent && body != "" {
			t.Errorf("%s %s: ответ 204 должен быть без тела, получено %s", test.method, test.path, body)
		}
	}
}

func TestLabThreeAPISeparatesListAndFeedFlags(t *testing.T) {
	server := newAPITestServer()
	cases := []struct{ path, contains, absent string }{
		{"/api/riscv_commands", `"creator_login":`, `"is_liked"`},
		{"/api/riscv_commands/feed", `"creator_login":`, `"is_creator"`},
	}
	for _, test := range cases {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		body := response.Body.String()
		if response.Code != http.StatusOK || !strings.Contains(body, test.contains) || strings.Contains(body, test.absent) {
			t.Errorf("GET %s: код %d, тело %s", test.path, response.Code, body)
		}
	}
}
