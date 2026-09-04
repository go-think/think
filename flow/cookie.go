package flow

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

// DefaultCookieConfigProvider allows injecting a configuration loader without import cycle
var DefaultCookieConfigProvider func() *CookieConfig

type CookieConfig struct {
	Prefix          string
	Path            string        // optional
	Domain          string        // optional
	ExpiresDuration time.Duration // Expiration duration configuration
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

		cookie = &http.Cookie{
			Name:  v,
			Value: url.QueryEscape(params[0].(string)),
		}

		if len(params) > 1 {
			switch p := params[1].(type) {
			case int:
				cookie.MaxAge = p
			case time.Duration:
				cookie.Expires = time.Now().Add(p)
			case time.Time:
				cookie.Expires = p
			}
		}

		if len(params) > 2 {
			cookie.Path = params[2].(string)
		}

		if len(params) > 3 {
			cookie.Domain = params[3].(string)
		}

		if len(params) > 4 {
			cookie.Secure = params[4].(bool)
		}

		if len(params) > 5 {
			cookie.HttpOnly = params[5].(bool)
		}

		if c.Config != nil {
			if cookie.Path == "" {
				cookie.Path = c.Config.Path
			}
			if cookie.Domain == "" {
				cookie.Domain = c.Config.Domain
			}
			if !cookie.Secure && c.Config.Secure {
				cookie.Secure = c.Config.Secure
			}
			if !cookie.HttpOnly && c.Config.HttpOnly {
				cookie.HttpOnly = c.Config.HttpOnly
			}
			if cookie.Expires.IsZero() && c.Config.ExpiresDuration > 0 {
				cookie.Expires = time.Now().Add(c.Config.ExpiresDuration)
			}
		}
	default:
		return nil, errors.New("Invalid parameters for Cookie.")
	}
	return cookie, nil
}

func ParseCookieHandler() *Cookie {
	if DefaultCookieConfigProvider != nil {
		if cfg := DefaultCookieConfigProvider(); cfg != nil {
			return &Cookie{Config: cfg}
		}
	}

	return &Cookie{
		Config: &CookieConfig{
			Prefix:          "",
			Path:            "/",
			Domain:          "",
			ExpiresDuration: time.Hour * 4,
			MaxAge:          0,
			Secure:          false,
			HttpOnly:        true,
		},
	}
}
