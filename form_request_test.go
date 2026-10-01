package think

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-think/flow"
	"github.com/stretchr/testify/assert"
)

// Example FormRequest
type CreateUserTestRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	IsAdmin  bool   `json:"is_admin"`
}

func (r *CreateUserTestRequest) Rules() map[string]string {
	return map[string]string{
		"name":  "required|min:3",
		"email": "required|email",
		"age":   "required|integer|min:18",
	}
}

// Example FormRequest with Authorization check
type UnauthorizedTestRequest struct {
	Title string `json:"title"`
}

func (r *UnauthorizedTestRequest) Authorize() bool {
	return false
}

func (r *UnauthorizedTestRequest) Rules() map[string]string {
	return map[string]string{
		"title": "required",
	}
}

func TestFormRequest_AutoValidation_Success(t *testing.T) {
	app := Configure().
		WithRouting(Routing{
			Api: func(r flow.Router) {
				r.Post("/users", func(req *CreateUserTestRequest) any {
					return flow.Json(map[string]any{
						"created": true,
						"name":    req.Name,
						"email":   req.Email,
						"age":     req.Age,
					})
				})
			},
		}).
		Create()

	server := httptest.NewServer(app.BuildServer().Handler)
	defer server.Close()

	// Valid payload
	body := `{"name":"Taylor","email":"taylor@example.com","age":30}`
	resp, err := http.Post(server.URL+"/api/users", "application/json", strings.NewReader(body))
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var data map[string]any
	err = json.Unmarshal(respBody, &data)
	assert.NoError(t, err)
	assert.Equal(t, true, data["created"])
	assert.Equal(t, "Taylor", data["name"])
	assert.Equal(t, "taylor@example.com", data["email"])
	assert.Equal(t, float64(30), data["age"])
}

func TestFormRequest_AutoValidation_Fails422(t *testing.T) {
	called := false
	app := Configure().
		WithRouting(Routing{
			Api: func(r flow.Router) {
				r.Post("/users", func(req *CreateUserTestRequest) any {
					called = true
					return flow.Json(map[string]any{"ok": true})
				})
			},
		}).
		Create()

	server := httptest.NewServer(app.BuildServer().Handler)
	defer server.Close()

	// Invalid payload (name too short, invalid email, age under 18)
	body := `{"name":"T","email":"invalid-email","age":12}`
	resp, err := http.Post(server.URL+"/api/users", "application/json", strings.NewReader(body))
	assert.NoError(t, err)
	assert.Equal(t, 422, resp.StatusCode)
	assert.False(t, called, "Controller action should not execute when validation fails")

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var errResp map[string]any
	err = json.Unmarshal(respBody, &errResp)
	assert.NoError(t, err)
	assert.Equal(t, "The given data was invalid.", errResp["message"])

	errorsMap, ok := errResp["errors"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, errorsMap, "name")
	assert.Contains(t, errorsMap, "email")
	assert.Contains(t, errorsMap, "age")
}

func TestFormRequest_Authorization_Fails403(t *testing.T) {
	called := false
	app := Configure().
		WithRouting(Routing{
			Api: func(r flow.Router) {
				r.Post("/admin/secret", func(req *UnauthorizedTestRequest) any {
					called = true
					return flow.Json(map[string]any{"ok": true})
				})
			},
		}).
		Create()

	server := httptest.NewServer(app.BuildServer().Handler)
	defer server.Close()

	body := `{"title":"Classified"}`
	resp, err := http.Post(server.URL+"/api/admin/secret", "application/json", strings.NewReader(body))
	assert.NoError(t, err)
	assert.Equal(t, 403, resp.StatusCode)
	assert.False(t, called, "Controller action should not execute when authorization fails")
}

// Embedded Struct Test
type PaginationRequest struct {
	Page    int `form:"page"`
	PerPage int `form:"per_page"`
}

type UpdateUserWithRouteParamsRequest struct {
	PaginationRequest
	ID   int    `route:"id"`
	Role string `json:"role"`
}

func (r *UpdateUserWithRouteParamsRequest) Rules() map[string]string {
	return map[string]string{
		"id":   "required|integer|min:1",
		"role": "required|in:admin,editor",
	}
}

func TestFormRequest_RouteParamAndEmbeddedStruct(t *testing.T) {
	app := Configure().
		WithRouting(Routing{
			Api: func(r flow.Router) {
				r.Put("/users/{id}", func(req *UpdateUserWithRouteParamsRequest) any {
					return flow.Json(map[string]any{
						"id":       req.ID,
						"role":     req.Role,
						"page":     req.Page,
						"per_page": req.PerPage,
					})
				})
			},
		}).
		Create()

	server := httptest.NewServer(app.BuildServer().Handler)
	defer server.Close()

	// 1. Valid request with route param {id}=42, query ?page=2&per_page=15, and JSON body {"role":"admin"}
	reqBody := `{"role":"admin"}`
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/users/42?page=2&per_page=15", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var data map[string]any
	err = json.Unmarshal(respBody, &data)
	assert.NoError(t, err)
	assert.Equal(t, float64(42), data["id"])
	assert.Equal(t, "admin", data["role"])
	assert.Equal(t, float64(2), data["page"])
	assert.Equal(t, float64(15), data["per_page"])

	// 2. Invalid role -> 422
	badReqBody := `{"role":"superuser"}`
	badReq, _ := http.NewRequest(http.MethodPut, server.URL+"/api/users/42", strings.NewReader(badReqBody))
	badReq.Header.Set("Content-Type", "application/json")

	badResp, err := http.DefaultClient.Do(badReq)
	assert.NoError(t, err)
	assert.Equal(t, 422, badResp.StatusCode)
	badResp.Body.Close()
}
