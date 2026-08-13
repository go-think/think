package context

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/go-think/think/config"
)

type CookieConfig struct {
	Prefix          string
	Path            string        // optional
	Domain          string        // optional
	ExpiresDuration time.Duration // 过期时长配置
	RawExpires      string        // for reading cookies only

	MaxAge   int
	Secure   bool
	HttpOnly bool
	Raw      string
	Unparsed []string
}

type Cookie struct {
	Config *CookieConfig
}

func (c *Cookie) Set(name interface{}, params ...interface{}) (*http.Cookie, error) {
	var cookie *http.Cookie

	switch v := name.(type) {
	case *http.Cookie:
		cookie = v
	case string:
		if len(params) == 0 {
			return nil, errors.New("Invalid parameters for Cookie.")
		}

		valStr, ok := params[0].(string)
		if !ok {
			return nil, errors.New("Invalid parameters for Cookie.")
		}

		expires := time.Time{}
		if c.Config.ExpiresDuration > 0 {
			expires = time.Now().Add(c.Config.ExpiresDuration)
		}

		cookie = &http.Cookie{
			Name:     c.Config.Prefix + v,
			Value:    url.QueryEscape(valStr),
			Path:     c.Config.Path,
			Domain:   c.Config.Domain,
			Expires:  expires,
			MaxAge:   c.Config.MaxAge,
			Secure:   c.Config.Secure,
			HttpOnly: c.Config.HttpOnly,
		}

		if len(params) > 1 {
			if maxAge, ok := params[1].(int); ok {
				cookie.MaxAge = maxAge
			}
		}

		if len(params) > 2 {
			if pathStr, ok := params[2].(string); ok {
				cookie.Path = pathStr
			}
		}

		if len(params) > 3 {
			if domainStr, ok := params[3].(string); ok {
				cookie.Domain = domainStr
			}
		}

		if len(params) > 4 {
			if secureBool, ok := params[4].(bool); ok {
				cookie.Secure = secureBool
			}
		}
	default:
		return nil, errors.New("Invalid parameters for Cookie.")
	}
	return cookie, nil
}

func ParseCookieHandler() *Cookie {
	return &Cookie{
		Config: &CookieConfig{
			Prefix:          config.Cookie.Prefix,
			Path:            config.Cookie.Path,
			Domain:          config.Cookie.Domain,
			ExpiresDuration: config.Cookie.Expires,
			MaxAge:          config.Cookie.MaxAge,
			Secure:          config.Cookie.Secure,
			HttpOnly:        config.Cookie.HttpOnly,
		},
	}
}
