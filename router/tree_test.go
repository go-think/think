package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRadixTree_BasicAndParams(t *testing.T) {
	root := &node{}

	root.addRoute("/", "root_handler")
	root.addRoute("/user/:id", "user_detail")
	root.addRoute("/user/:id/posts", "user_posts")
	root.addRoute("/static/*filepath", "static_handler")

	// 1. 测试根节点匹配
	h, ps, tsr := root.getValue("/")
	assert.Equal(t, "root_handler", h)
	assert.False(t, tsr)
	assert.Empty(t, ps)

	// 2. 测试命名参数提取
	h, ps, tsr = root.getValue("/user/123")
	assert.Equal(t, "user_detail", h)
	assert.False(t, tsr)
	val, ok := ps.Get("id")
	assert.True(t, ok)
	assert.Equal(t, "123", val)

	// 3. 测试深层命名参数
	h, ps, tsr = root.getValue("/user/456/posts")
	assert.Equal(t, "user_posts", h)
	val, ok = ps.Get("id")
	assert.True(t, ok)
	assert.Equal(t, "456", val)

	// 4. 测试通配符匹配
	h, ps, tsr = root.getValue("/static/css/style.css")
	assert.Equal(t, "static_handler", h)
	val, ok = ps.Get("filepath")
	assert.True(t, ok)
	assert.Equal(t, "css/style.css", val)
}
