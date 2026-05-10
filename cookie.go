package chrome

import (
	"time"
)

// Cookie represents an HTTP cookie.
//
// Upstream equivalent: HeadlessChromium\Cookies\Cookie
type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  time.Time
	HTTPOnly bool
	Secure   bool
	SameSite string
}

// CookieOptions configures a new cookie.
type CookieOptions struct {
	Domain   string
	Path     string
	Expires  time.Time
	HTTPOnly bool
	Secure   bool
	SameSite string
}

// NewCookie creates a Cookie with the given name, value, and options.
//
// Upstream equivalent: new Cookie($name, $value, $opts).
func NewCookie(name, value string, opts CookieOptions) Cookie {
	return Cookie{
		Name:     name,
		Value:    value,
		Domain:   opts.Domain,
		Path:     opts.Path,
		Expires:  opts.Expires,
		HTTPOnly: opts.HTTPOnly,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	}
}

// CookieList is a slice of cookies with helper filter methods.
//
// Upstream equivalent: HeadlessChromium\Cookies\CookieCollection
type CookieList []Cookie

// FilterBy returns all cookies where the given field equals value.
// Supported fields: "name", "value", "domain", "path", "samesite".
func (c CookieList) FilterBy(field, value string) CookieList {
	var out CookieList
	for _, cookie := range c {
		if cookieFieldMatch(cookie, field, value) {
			out = append(out, cookie)
		}
	}
	return out
}

// FindOneBy returns the first cookie where the given field equals value.
func (c CookieList) FindOneBy(field, value string) (Cookie, bool) {
	for _, cookie := range c {
		if cookieFieldMatch(cookie, field, value) {
			return cookie, true
		}
	}
	return Cookie{}, false
}

func cookieFieldMatch(c Cookie, field, value string) bool {
	switch field {
	case "name":
		return c.Name == value
	case "value":
		return c.Value == value
	case "domain":
		return c.Domain == value
	case "path":
		return c.Path == value
	case "samesite":
		return c.SameSite == value
	}
	return false
}

// cookieRaw is the CDP wire format for a cookie.
type cookieRaw struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

func rawToCookie(r cookieRaw) Cookie {
	var expires time.Time
	if r.Expires > 0 {
		expires = time.Unix(int64(r.Expires), 0)
	}
	return Cookie{
		Name:     r.Name,
		Value:    r.Value,
		Domain:   r.Domain,
		Path:     r.Path,
		Expires:  expires,
		HTTPOnly: r.HTTPOnly,
		Secure:   r.Secure,
		SameSite: r.SameSite,
	}
}

func cookieToParams(c Cookie) map[string]any {
	params := map[string]any{
		"name":     c.Name,
		"value":    c.Value,
		"domain":   c.Domain,
		"path":     c.Path,
		"httpOnly": c.HTTPOnly,
		"secure":   c.Secure,
	}
	if !c.Expires.IsZero() {
		params["expires"] = float64(c.Expires.Unix())
	}
	if c.SameSite != "" {
		params["sameSite"] = c.SameSite
	}
	return params
}
