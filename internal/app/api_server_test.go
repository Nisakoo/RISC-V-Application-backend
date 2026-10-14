package app

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"riscv-lab/internal/domain"
	"riscv-lab/internal/handler"
	"riscv-lab/internal/repository"
	"riscv-lab/internal/service"
)

const testUserPassword = "riscv-test-password"

type likeKey struct{ userID, commandID int64 }

type apiApplicationRepository struct {
	*applicationRepository
	users map[string]domain.User
	liked map[likeKey]bool
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

func (fake *apiApplicationRepository) SetLike(_ context.Context, id, userID int64, liked bool) (int64, error) {
	for index := range fake.commands {
		if fake.commands[index].ID == id && fake.commands[index].Status == domain.RiscVCommandPublished {
			fake.liked[likeKey{userID, id}] = liked
			count := int64(0)
			for key, value := range fake.liked {
				if key.commandID == id && value {
					count++
				}
			}
			fake.commands[index].LikeCount = count
			return count, nil
		}
	}
	return 0, repository.ErrRiscVCommandNotFound
}

func (fake *apiApplicationRepository) IsLiked(_ context.Context, id, userID int64) (bool, error) {
	return fake.liked[likeKey{userID, id}], nil
}

func (fake *apiApplicationRepository) RegisterUser(_ context.Context, login, passwordHash string) (int64, error) {
	if _, exists := fake.users[login]; exists {
		return 0, repository.ErrLoginAlreadyExists
	}
	id := int64(len(fake.users) + 10)
	fake.users[login] = domain.User{ID: id, Login: login, PasswordHash: passwordHash, Role: domain.UserRoleUser}
	return id, nil
}

func (fake *apiApplicationRepository) GetUserByLogin(_ context.Context, login string) (domain.User, error) {
	user, exists := fake.users[login]
	if !exists {
		return domain.User{}, repository.ErrUserNotFound
	}
	return user, nil
}

// memoryBlacklist хранит отозванные JWT в памяти для тестов без Redis
type memoryBlacklist map[string]bool

func (blacklist memoryBlacklist) Add(_ context.Context, tokenID, _ string, _ time.Duration) error {
	blacklist[tokenID] = true
	return nil
}

func (blacklist memoryBlacklist) Contains(_ context.Context, tokenID string) (bool, error) {
	return blacklist[tokenID], nil
}

type apiApplicationMedia struct{ applicationMedia }

func (apiApplicationMedia) Upload(_ context.Context, kind string, _ *multipart.FileHeader) (string, error) {
	return kind + "/test.png", nil
}

func (apiApplicationMedia) Remove(_ context.Context, _ string) error { return nil }

func newAPITestServer(t *testing.T) *Server {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testUserPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	fake := &apiApplicationRepository{
		applicationRepository: &applicationRepository{commands: []domain.RiscVCommand{
			{ID: 1, Mnemonic: "ADD", Status: domain.RiscVCommandPublished, CreatorID: 1, CreatorLogin: "riscv_moderator", OperandCount: 3},
			{ID: 2, Mnemonic: "SUB", Status: domain.RiscVCommandPublished, CreatorID: 2, CreatorLogin: "pipeline_author", OperandCount: 3},
			{ID: 3, Mnemonic: "OR", Status: domain.RiscVCommandDeleted, CreatorID: 1, CreatorLogin: "riscv_moderator", OperandCount: 3},
		}},
		users: map[string]domain.User{
			"riscv_moderator": {ID: 1, Login: "riscv_moderator", PasswordHash: string(hash), Role: domain.UserRoleModerator},
			"pipeline_author": {ID: 2, Login: "pipeline_author", PasswordHash: string(hash), Role: domain.UserRoleUser},
		},
		liked: map[likeKey]bool{},
	}
	media := apiApplicationMedia{}
	userService := service.NewUserService(fake, memoryBlacklist{}, []byte(strings.Repeat("k", 32)), time.Hour)
	return NewServer(":0", handler.NewRiscVCommandHandler(service.NewRiscVCommandService(fake), media), &API{
		RiscVCommands: handler.NewRiscVAPIHandler(service.NewRiscVAPIService(fake), media),
		Users:         handler.NewUserAPIHandler(userService),
		Tokens:        userService,
	})
}

func serve(server *Server, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	return response
}

func authenticate(t *testing.T, server *Server, login string) string {
	t.Helper()
	response := serve(server, http.MethodPost, "/api/users/authentication", `{"login":"`+login+`","password":"`+testUserPassword+`"}`, "")
	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Role        string `json:"role"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &result) != nil || result.AccessToken == "" || result.TokenType != "Bearer" || result.Role == "" {
		t.Fatalf("аутентификация %s: код %d, тело %s", login, response.Code, response.Body.String())
	}
	return result.AccessToken
}

func TestLabFourHasTenAPIMethods(t *testing.T) {
	server := newAPITestServer(t)
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

func TestLabFourAPIHidesDeletedAndValidatesInput(t *testing.T) {
	server := newAPITestServer(t)
	token := authenticate(t, server, "riscv_moderator")
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
		response := serve(server, test.method, test.path, test.body, token)
		if response.Code != test.status || !strings.Contains(response.Body.String(), test.contains) || (test.absent != "" && strings.Contains(response.Body.String(), test.absent)) {
			t.Errorf("%s %s: код %d, тело %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}

func TestLabFourAPIReturnsDataWithoutStatusWrapper(t *testing.T) {
	server := newAPITestServer(t)
	token := authenticate(t, server, "riscv_moderator")
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
		response := serve(server, test.method, test.path, test.body, token)
		body := response.Body.String()
		if response.Code != test.status || !strings.HasPrefix(body, test.prefix) || !strings.Contains(body, test.contains) || strings.Contains(body, `"data"`) || strings.Contains(body, `"success"`) || strings.Contains(body, `"creator":{`) {
			t.Errorf("%s %s: код %d, тело %s", test.method, test.path, response.Code, body)
		}
		if test.status == http.StatusNoContent && body != "" {
			t.Errorf("%s %s: ответ 204 должен быть без тела, получено %s", test.method, test.path, body)
		}
	}
}

func TestLabFourAPISeparatesListAndFeedFlags(t *testing.T) {
	server := newAPITestServer(t)
	cases := []struct{ path, contains, absent string }{
		{"/api/riscv_commands", `"creator_login":`, `"is_liked"`},
		{"/api/riscv_commands/feed", `"creator_login":`, `"is_creator"`},
	}
	for _, test := range cases {
		response := serve(server, http.MethodGet, test.path, "", "")
		body := response.Body.String()
		if response.Code != http.StatusOK || !strings.Contains(body, test.contains) || strings.Contains(body, test.absent) {
			t.Errorf("GET %s: код %d, тело %s", test.path, response.Code, body)
		}
	}
}

func TestLabFourGuestReadsButCannotChangeCommands(t *testing.T) {
	server := newAPITestServer(t)
	for _, token := range []string{"", "not-a-jwt"} {
		if response := serve(server, http.MethodGet, "/api/riscv_commands", "", token); response.Code != http.StatusOK || strings.Contains(response.Body.String(), `"is_creator":1`) {
			t.Errorf("список для гостя с токеном %q: код %d, тело %s", token, response.Code, response.Body.String())
		}
		if response := serve(server, http.MethodGet, "/api/riscv_commands/feed", "", token); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"is_liked":0`) {
			t.Errorf("лента для гостя с токеном %q: код %d, тело %s", token, response.Code, response.Body.String())
		}
	}
	protected := []struct{ method, path, body string }{
		{http.MethodGet, "/api/riscv_commands/draft", ""},
		{http.MethodPost, "/api/riscv_commands", ""},
		{http.MethodPut, "/api/riscv_commands/1/publish", `{"description":"Достаточное описание","operand_count":3,"length_bits":32}`},
		{http.MethodDelete, "/api/riscv_commands/1", ""},
		{http.MethodPost, "/api/riscv_commands/1/like", `{"like":1}`},
		{http.MethodPost, "/api/users/logout", ""},
	}
	for _, test := range protected {
		response := serve(server, test.method, test.path, test.body, "")
		if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"message":`) {
			t.Errorf("гость %s %s: код %d, тело %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}

func TestLabFourCreatorComesFromToken(t *testing.T) {
	server := newAPITestServer(t)
	token := authenticate(t, server, "pipeline_author")
	response := serve(server, http.MethodGet, "/api/riscv_commands", "", token)
	var commands []struct {
		ID        int64 `json:"id"`
		IsCreator int   `json:"is_creator"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &commands); err != nil {
		t.Fatal(err)
	}
	for _, command := range commands {
		if (command.ID == 2) != (command.IsCreator == 1) {
			t.Errorf("признак is_creator команды %d для pipeline_author: %d", command.ID, command.IsCreator)
		}
	}
	if response := serve(server, http.MethodDelete, "/api/riscv_commands/1", "", token); response.Code != http.StatusNotFound {
		t.Errorf("удаление чужой команды: код %d", response.Code)
	}
	if response := serve(server, http.MethodDelete, "/api/riscv_commands/2", "", token); response.Code != http.StatusNoContent {
		t.Errorf("удаление своей команды: код %d, тело %s", response.Code, response.Body.String())
	}
}

func TestLabFourAuthenticationAndLogout(t *testing.T) {
	server := newAPITestServer(t)
	if response := serve(server, http.MethodPost, "/api/users/authentication", `{"login":"pipeline_author","password":"wrong-password"}`, ""); response.Code != http.StatusUnauthorized {
		t.Errorf("неверный пароль: код %d", response.Code)
	}
	if response := serve(server, http.MethodPost, "/api/users/authentication", `{"login":"unknown_user","password":"wrong-password"}`, ""); response.Code != http.StatusUnauthorized {
		t.Errorf("неизвестный логин: код %d", response.Code)
	}
	if response := serve(server, http.MethodPost, "/api/users", `{"login":"rv64_author","password":"`+testUserPassword+`"}`, ""); response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("регистрация: код %d, тело %s", response.Code, response.Body.String())
	}
	token := authenticate(t, server, "rv64_author")
	if response := serve(server, http.MethodGet, "/api/riscv_commands/draft", "", token); response.Code != http.StatusNotFound {
		t.Errorf("черновик нового пользователя: код %d, тело %s", response.Code, response.Body.String())
	}
	if response := serve(server, http.MethodPost, "/api/users/logout", "", token); response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("деавторизация: код %d, тело %s", response.Code, response.Body.String())
	}
	if response := serve(server, http.MethodGet, "/api/riscv_commands/draft", "", token); response.Code != http.StatusUnauthorized {
		t.Errorf("отозванный токен принят: код %d", response.Code)
	}
}

func TestLabFourServesSwagger(t *testing.T) {
	server := newAPITestServer(t)
	response := serve(server, http.MethodGet, "/swagger/doc.json", "", "")
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, `"BearerAuth"`) || !strings.Contains(body, `/api/users/authentication`) {
		t.Fatalf("спецификация Swagger: код %d", response.Code)
	}
}
