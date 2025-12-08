package context

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/go-think/think/config"
)

type CookieConfig struct {
	Prefix string

	Path       string    // optional
	Domain     string    // optional
	Expires    time.Time // optional
	RawExpires string    // for reading cookies only

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	MaxAge   int
	Secure   bool
	HttpOnly bool
	Raw      string
	Unparsed []string // Raw text of unparsed attribute-value pairs
}

type Cookie struct {
	Config *CookieConfig
}

func (c *Cookie) Set(name interface{}, params ...interface{}) (*http.Cookie, error) {
	var cookie *http.Cookie

	switch name.(type) {
	case *http.Cookie:
		cookie = name.(*http.Cookie)
	case string:
		if len(params) == 0 {
			return nil, errors.New("Invalid parameters for Cookie.")
		}

		value := params[0]
		if _, ok := value.(string); !ok {
			return nil, errors.New("Invalid parameters for Cookie.")
		}
		cookie = &http.Cookie{
			Name:     c.Config.Prefix + name.(string),
			Value:    url.QueryEscape(value.(string)),
			Path:     c.Config.Path,
			Domain:   c.Config.Domain,
			Expires:  c.Config.Expires,
			MaxAge:   c.Config.MaxAge,
			Secure:   c.Config.Secure,
			HttpOnly: c.Config.HttpOnly,
		}

		if len(params) > 1 {
			maxAge := params[1]
			if _, ok := maxAge.(int); !ok {
				return nil, errors.New("Invalid parameters for Cookie.")
			}
			cookie.MaxAge = maxAge.(int)
		}

		if len(params) > 2 {
			path := params[2]
			if _, ok := path.(string); !ok {
				return nil, errors.New("Invalid parameters for Cookie.")
			}
			cookie.Path = path.(string)
		}

		if len(params) > 3 {
			domain := params[3]
			if _, ok := domain.(string); !ok {
				return nil, errors.New("Invalid parameters for Cookie.")
			}
			cookie.Domain = domain.(string)
		}

		if len(params) > 4 {
			secure := params[4]
			if _, ok := secure.(bool); !ok {
				return nil, errors.New("Invalid parameters for Cookie.")
			}
			cookie.Secure = secure.(bool)
		}
	default:
		return nil, errors.New("Invalid parameters for Cookie.")
	}
	return cookie, nil
}

func ParseCookieHandler() *Cookie {
	return &Cookie{
		Config: &CookieConfig{
			Prefix:   config.Cookie.Prefix,
			Path:     config.Cookie.Path,
			Domain:   config.Cookie.Domain,
			Expires:  time.Now().Add(config.Cookie.Expires),
			MaxAge:   config.Cookie.MaxAge,
			Secure:   config.Cookie.Secure,
			HttpOnly: config.Cookie.HttpOnly,
		},
	}
}
