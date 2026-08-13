package think

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	contextPkg "github.com/go-think/think/context"
	"github.com/stretchr/testify/assert"
)

func testRequest(t *testing.T, method, reqUrl string, data url.Values, res *Res) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}

	var body io.Reader
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		if data != nil {
			reqUrl = strings.TrimRight(reqUrl, "?") + "?" + data.Encode()
		}
	case "POST", "PUT", "DELETE":
		if data != nil {
			body = strings.NewReader(data.Encode())
		}
	}

	req, err := http.NewRequest(method, reqUrl, body)
	assert.NoError(t, err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	content, ioerr := io.ReadAll(resp.Body)
	assert.NoError(t, ioerr)

	assert.Equal(t, res.GetCode(), resp.StatusCode)
	assert.Equal(t, res.GetContent(), string(content))
}

func TestRunWithPort(t *testing.T) {
	th := New()

	go func() {
		th.RegisterRoute(func(route *Route) {
			route.Get("/", func(req *Req) *Res {
				return Text("it worked")
			})
		})
		th.Run(":9012")
	}()

	time.Sleep(300 * time.Millisecond)

	testRequest(t, "get", "http://localhost:9012/", nil, contextPkg.NewResponse().SetContent("it worked"))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = th.Shutdown(ctx)
}

func TestThink_Run(t *testing.T) {
	th := New()

	go func() {
		th.RegisterRoute(func(route *Route) {
			route.Get("/", func(req *Req) interface{} {
				return "it worked"
			})
			route.Get("/user/{name}", func(req *Req, name string) interface{} {
				return fmt.Sprintf("Hello %s !", name)
			})
			route.Post("/user", func(req *Req) interface{} {
				name, err := req.Post("name")
				if err != nil {
					panic(err)
				}
				return fmt.Sprintf("Create %s", name)
			})
			route.Delete("/user/{name}", func(name string) interface{} {
				return fmt.Sprintf("Delete %s", name)
			})
		})
		th.Run(":9011")
	}()

	time.Sleep(300 * time.Millisecond)

	testRequest(t, "get", "http://localhost:9011/", nil, contextPkg.NewResponse().SetContent("it worked"))
	testRequest(t, "get", "http://localhost:9011/user/thinkgo", nil, contextPkg.NewResponse().SetContent(fmt.Sprintf("Hello %s !", "thinkgo")))
	testRequest(t, "post", "http://localhost:9011/user", url.Values{"name": {"thinkgo"}}, contextPkg.NewResponse().SetContent(fmt.Sprintf("Create %s", "thinkgo")))
	testRequest(t, "delete", "http://localhost:9011/user/thinkgo", url.Values{"name": {"thinkgo"}}, contextPkg.NewResponse().SetContent(fmt.Sprintf("Delete %s", "thinkgo")))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = th.Shutdown(ctx)
}

