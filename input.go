package chrome

import (
	"fmt"
	"time"

	"github.com/masfu/chrome-go/cdp"
)

// Mouse button constants.
const (
	MouseButtonLeft   = "left"
	MouseButtonRight  = "right"
	MouseButtonMiddle = "middle"
)

// MoveOption configures a mouse move action.
type MoveOption func(*moveConfig)

type moveConfig struct {
	steps int
}

// WithMoveSteps sets the number of intermediate steps for a mouse move.
func WithMoveSteps(steps int) MoveOption {
	return func(c *moveConfig) { c.steps = steps }
}

// ClickOption configures a mouse click action.
type ClickOption func(*clickConfig)

type clickConfig struct {
	button     string
	clickCount int
}

// WithButton sets which mouse button to click.
func WithButton(button string) ClickOption {
	return func(c *clickConfig) { c.button = button }
}

// WithDoubleClick configures the click as a double-click.
func WithDoubleClick() ClickOption {
	return func(c *clickConfig) { c.clickCount = 2 }
}

// Mouse provides mouse input simulation for a page.
//
// Upstream equivalent: HeadlessChromium\Input\Mouse
type Mouse struct {
	page *Page
	x    float64
	y    float64
}

// Move moves the mouse cursor to the given coordinates.
//
// Upstream equivalent: $mouse->move($x, $y).
func (m *Mouse) Move(x, y int, opts ...MoveOption) *Mouse {
	cfg := &moveConfig{steps: 1}
	for _, o := range opts {
		o(cfg)
	}

	dx := (float64(x) - m.x) / float64(cfg.steps)
	dy := (float64(y) - m.y) / float64(cfg.steps)

	ctx, cancel := withDefaultTimeout(m.page)
	defer cancel()

	for i := 0; i < cfg.steps; i++ {
		m.x += dx
		m.y += dy
		m.page.session.SendMessageSync(ctx, cdp.Message{ //nolint:errcheck
			Method: "Input.dispatchMouseEvent",
			Params: map[string]any{
				"type": "mouseMoved",
				"x":    m.x,
				"y":    m.y,
			},
		})
	}
	return m
}

// Click dispatches a mouse click at the current position.
//
// Upstream equivalent: $mouse->click($opts).
func (m *Mouse) Click(opts ...ClickOption) *Mouse {
	cfg := &clickConfig{button: MouseButtonLeft, clickCount: 1}
	for _, o := range opts {
		o(cfg)
	}

	ctx, cancel := withDefaultTimeout(m.page)
	defer cancel()

	for _, t := range []string{"mousePressed", "mouseReleased"} {
		m.page.session.SendMessageSync(ctx, cdp.Message{ //nolint:errcheck
			Method: "Input.dispatchMouseEvent",
			Params: map[string]any{
				"type":       t,
				"x":          m.x,
				"y":          m.y,
				"button":     cfg.button,
				"clickCount": cfg.clickCount,
			},
		})
	}
	return m
}

// ScrollUp scrolls up by the given number of pixels.
//
// Upstream equivalent: $mouse->scrollUp($px).
func (m *Mouse) ScrollUp(px int) *Mouse {
	m.scroll(0, -px)
	return m
}

// ScrollDown scrolls down by the given number of pixels.
//
// Upstream equivalent: $mouse->scrollDown($px).
func (m *Mouse) ScrollDown(px int) *Mouse {
	m.scroll(0, px)
	return m
}

func (m *Mouse) scroll(deltaX, deltaY int) {
	ctx, cancel := withDefaultTimeout(m.page)
	defer cancel()
	m.page.session.SendMessageSync(ctx, cdp.Message{ //nolint:errcheck
		Method: "Input.dispatchMouseEvent",
		Params: map[string]any{
			"type":   "mouseWheel",
			"x":      m.x,
			"y":      m.y,
			"deltaX": deltaX,
			"deltaY": deltaY,
		},
	})
}

// Find moves the mouse to the center of the element matching selector.
// An optional nth index (0-based) selects among multiple matches.
//
// Upstream equivalent: $mouse->find($selector, $nth).
func (m *Mouse) Find(selector string, nth ...int) (*Element, error) {
	dom := m.page.DOM()
	elements, err := dom.QuerySelectorAll(selector)
	if err != nil {
		return nil, err
	}
	idx := 0
	if len(nth) > 0 {
		idx = nth[0]
	}
	if idx >= len(elements) {
		return nil, fmt.Errorf("%w: selector %q index %d", ErrElementNotFound, selector, idx)
	}
	el := elements[idx]
	box, err := el.boundingBox()
	if err != nil {
		return nil, err
	}
	m.Move(int(box.x+box.width/2), int(box.y+box.height/2))
	return el, nil
}

// Keyboard provides keyboard input simulation for a page.
//
// Upstream equivalent: HeadlessChromium\Input\Keyboard
type Keyboard struct {
	page        *Page
	keyInterval time.Duration
}

// SetKeyInterval sets the delay between key events.
//
// Upstream equivalent: $keyboard->setKeyInterval($d).
func (k *Keyboard) SetKeyInterval(d time.Duration) *Keyboard {
	k.keyInterval = d
	return k
}

// TypeRawKey dispatches a raw key event by key code.
//
// Upstream equivalent: $keyboard->typeRawKey($key).
func (k *Keyboard) TypeRawKey(key string) *Keyboard {
	k.Press(key)
	return k
}

// TypeText types a sequence of characters.
//
// Upstream equivalent: $keyboard->typeText($text).
func (k *Keyboard) TypeText(text string) *Keyboard {
	ctx, cancel := withDefaultTimeout(k.page)
	defer cancel()
	for _, ch := range text {
		k.page.session.SendMessageSync(ctx, cdp.Message{ //nolint:errcheck
			Method: "Input.dispatchKeyEvent",
			Params: map[string]any{
				"type": "char",
				"text": string(ch),
			},
		})
		if k.keyInterval > 0 {
			time.Sleep(k.keyInterval)
		}
	}
	return k
}

// Press dispatches a keyDown event.
//
// Upstream equivalent: $keyboard->press($key).
func (k *Keyboard) Press(key string) *Keyboard {
	k.dispatchKey("keyDown", normalizeKey(key))
	return k
}

// Type dispatches a keyDown followed by a keyUp event (press + release).
//
// Upstream equivalent: $keyboard->type($key).
func (k *Keyboard) Type(key string) *Keyboard {
	norm := normalizeKey(key)
	k.dispatchKey("keyDown", norm)
	if k.keyInterval > 0 {
		time.Sleep(k.keyInterval)
	}
	k.dispatchKey("keyUp", norm)
	return k
}

// Release dispatches keyUp events for the given keys. If no keys are given,
// behaviour is a no-op (Go has no concept of "release all pressed keys").
//
// Upstream equivalent: $keyboard->release($keys...).
func (k *Keyboard) Release(keys ...string) *Keyboard {
	for _, key := range keys {
		k.dispatchKey("keyUp", normalizeKey(key))
	}
	return k
}

func (k *Keyboard) dispatchKey(eventType, key string) {
	ctx, cancel := withDefaultTimeout(k.page)
	defer cancel()
	k.page.session.SendMessageSync(ctx, cdp.Message{ //nolint:errcheck
		Method: "Input.dispatchKeyEvent",
		Params: map[string]any{
			"type": eventType,
			"key":  key,
		},
	})
}

// normalizeKey maps common aliases to their canonical key names.
// NormalizeKey maps common key aliases ("ctrl", "cmd", "enter", etc.) to their
// standard DOM key names ("Control", "Meta", "Enter", etc.).
func NormalizeKey(key string) string { return normalizeKey(key) }

func normalizeKey(key string) string {
	aliases := map[string]string{
		"ctrl":       "Control",
		"cmd":        "Meta",
		"command":    "Meta",
		"alt":        "Alt",
		"shift":      "Shift",
		"enter":      "Enter",
		"return":     "Enter",
		"tab":        "Tab",
		"backspace":  "Backspace",
		"delete":     "Delete",
		"del":        "Delete",
		"escape":     "Escape",
		"esc":        "Escape",
		"arrowup":    "ArrowUp",
		"arrowdown":  "ArrowDown",
		"arrowleft":  "ArrowLeft",
		"arrowright": "ArrowRight",
		"home":       "Home",
		"end":        "End",
		"pageup":     "PageUp",
		"pagedown":   "PageDown",
		"space":      " ",
	}
	lower := key
	for k, v := range aliases {
		if lower == k {
			return v
		}
	}
	return key
}
