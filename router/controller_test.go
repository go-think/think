package router

import (
	"fmt"
	"net/http"
	"testing"
	"github.com/go-think/think/flow"
	"github.com/go-think/think/container"
	"github.com/go-think/think/facades"
	"github.com/go-think/think/contract"
	"github.com/stretchr/testify/assert"
)

type mockApp struct {
	*container.Container
	env string
}

func (m *mockApp) GetContainer() *container.Container {
	return m.Container
}

func (m *mockApp) Environment(envs ...string) bool { return true }
func (m *mockApp) IsProduction() bool { return false }
func (m *mockApp) IsLocal() bool { return true }
func (m *mockApp) IsTesting() bool { return true }
func (m *mockApp) SetEnvironment(env string) { m.env = env }
func (m *mockApp) GetEnvironment() string    { return m.env }
func (m *mockApp) IsDebug() bool             { return true }
func (m *mockApp) SetDebug(debug bool)       {}
func (m *mockApp) SetTimezone(tz string)               {}
func (m *mockApp) GetTimezone() string                 { return "UTC" }
func (m *mockApp) BasePath(path ...string) string      { return "" }
func (m *mockApp) SetBasePath(bp string) contract.Application { return m }
func (m *mockApp) ConfigPath(path ...string) string    { return "config" }
func (m *mockApp) StoragePath(path ...string) string   { return "storage" }
func (m *mockApp) PublicPath(path ...string) string    { return "public" }
func (m *mockApp) EnvironmentPath() string             { return "" }
func (m *mockApp) EnvironmentFile() string             { return ".env" }
func (m *mockApp) EnvironmentFilePath() string         { return ".env" }


type ConfigService struct {
	Greeting string
}

type UserController struct {
	Config *ConfigService
}

func (c *UserController) Index(req *flow.Request, id string) string {
	return fmt.Sprintf("%s user %s", c.Config.Greeting, id)
}

func TestControllerInjection(t *testing.T) {
	c := container.New()
	app := &mockApp{Container: c, env: "testing"}
	facades.App = app // Set global container for router to use

	// 1. Bind dependencies
	app.Singleton[*ConfigService](func() *ConfigService {
		return &ConfigService{Greeting: "Hello"}
	})

	// 2. Bind Controller as Singleton
	app.Singleton[*UserController](func() *UserController {
		return &UserController{
			Config: app.Make[*ConfigService](),
		}
	})

	// 3. Register Route
	r := New()
	r.Get("/user/{id}", (*UserController).Index)
	r.Register()

	// 4. Dispatch Request
	httpReq, _ := http.NewRequest("GET", "/user/123", nil)
	req := flow.NewRequest(httpReq)
	
	rule, params, err := r.MatchRequest(req)
	assert.NoError(t, err)
	assert.NotNil(t, rule)

	res := rule.Run(req, params)
	assert.Equal(t, "Hello user 123", res)
}
