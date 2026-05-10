package chrome_test

import (
	"testing"
	"time"

	chrome "github.com/masfu/chrome-go"
)

// ---- NewCookie ----

func TestNewCookie_FieldsSet(t *testing.T) {
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	c := chrome.NewCookie("session", "tok123", chrome.CookieOptions{
		Domain:   "example.com",
		Path:     "/api",
		Expires:  exp,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	if c.Name != "session" {
		t.Errorf("Name: want session, got %q", c.Name)
	}
	if c.Value != "tok123" {
		t.Errorf("Value: want tok123, got %q", c.Value)
	}
	if c.Domain != "example.com" {
		t.Errorf("Domain: want example.com, got %q", c.Domain)
	}
	if c.Path != "/api" {
		t.Errorf("Path: want /api, got %q", c.Path)
	}
	if !c.Expires.Equal(exp) {
		t.Errorf("Expires: want %v, got %v", exp, c.Expires)
	}
	if !c.HTTPOnly {
		t.Error("HTTPOnly should be true")
	}
	if !c.Secure {
		t.Error("Secure should be true")
	}
	if c.SameSite != "Strict" {
		t.Errorf("SameSite: want Strict, got %q", c.SameSite)
	}
}

// ---- CookieList.FilterBy ----

var testCookies = chrome.CookieList{
	{Name: "a", Value: "1", Domain: "foo.com", Path: "/", SameSite: "Lax"},
	{Name: "b", Value: "2", Domain: "bar.com", Path: "/admin", SameSite: "Strict"},
	{Name: "c", Value: "1", Domain: "foo.com", Path: "/other", SameSite: "Lax"},
}

func TestCookieList_FilterByName(t *testing.T) {
	got := testCookies.FilterBy("name", "a")
	if len(got) != 1 || got[0].Name != "a" {
		t.Errorf("FilterBy name=a: want 1 result with name a, got %v", got)
	}
}

func TestCookieList_FilterByValue(t *testing.T) {
	got := testCookies.FilterBy("value", "1")
	if len(got) != 2 {
		t.Errorf("FilterBy value=1: want 2 results, got %d", len(got))
	}
}

func TestCookieList_FilterByDomain(t *testing.T) {
	got := testCookies.FilterBy("domain", "foo.com")
	if len(got) != 2 {
		t.Errorf("FilterBy domain=foo.com: want 2 results, got %d", len(got))
	}
}

func TestCookieList_FilterByPath(t *testing.T) {
	got := testCookies.FilterBy("path", "/admin")
	if len(got) != 1 || got[0].Name != "b" {
		t.Errorf("FilterBy path=/admin: want [b], got %v", got)
	}
}

func TestCookieList_FilterBySameSite(t *testing.T) {
	got := testCookies.FilterBy("samesite", "Strict")
	if len(got) != 1 || got[0].Name != "b" {
		t.Errorf("FilterBy samesite=Strict: want [b], got %v", got)
	}
}

func TestCookieList_FilterByUnknownField(t *testing.T) {
	got := testCookies.FilterBy("nonexistent", "anything")
	if len(got) != 0 {
		t.Errorf("FilterBy unknown field: want 0 results, got %d", len(got))
	}
}

func TestCookieList_FilterByNoMatch(t *testing.T) {
	got := testCookies.FilterBy("name", "zzz")
	if len(got) != 0 {
		t.Errorf("FilterBy with no match: want 0, got %d", len(got))
	}
}

// ---- CookieList.FindOneBy ----

func TestCookieList_FindOneByFound(t *testing.T) {
	c, ok := testCookies.FindOneBy("name", "b")
	if !ok {
		t.Fatal("FindOneBy name=b: expected to find cookie")
	}
	if c.Domain != "bar.com" {
		t.Errorf("FindOneBy: wrong cookie returned, domain=%q", c.Domain)
	}
}

func TestCookieList_FindOneByNotFound(t *testing.T) {
	_, ok := testCookies.FindOneBy("name", "zzz")
	if ok {
		t.Error("FindOneBy: expected not found, got found")
	}
}

func TestCookieList_FindOneByReturnsFirst(t *testing.T) {
	// Multiple cookies with value "1" — should return the first one ("a").
	c, ok := testCookies.FindOneBy("value", "1")
	if !ok {
		t.Fatal("FindOneBy value=1: expected a result")
	}
	if c.Name != "a" {
		t.Errorf("FindOneBy: want first match (a), got %q", c.Name)
	}
}
