package session

type CookieHandler struct {
	request  Request
	response Response
}

func NewCookieHandler() *CookieHandler {
	return &CookieHandler{}
}

func (c *CookieHandler) SetRequest(req Request) {
	c.request = req
}

func (c *CookieHandler) SetResponse(res Response) {
	c.response = res
}

func (c *CookieHandler) Read(id string) string {
	if c.request == nil {
		return ""
	}
	value, _ := c.request.Cookie(id)

	return value
}

func (c *CookieHandler) Write(id string, data string) {
	if c.response == nil {
		return
	}
	c.response.Cookie(id, data)
}
