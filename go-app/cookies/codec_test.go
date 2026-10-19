package cookies

import (
	"strings"
	"testing"
)

type color string

type prefs struct {
	Theme string `json:"theme"`
	Size  int    `json:"size"`
}

func roundTrip[T any](t *testing.T, in T) T {
	t.Helper()
	enc, err := encodeCookieValue(in)
	if err != nil {
		t.Fatalf("encode %T: %v", in, err)
	}
	out, err := decodeCookieValue[T](enc)
	if err != nil {
		t.Fatalf("decode %T from %q: %v", in, enc, err)
	}
	return out
}

// Any pointer value used to panic in encode (interface conversion to T), and named
// primitive types panicked in decode.
func TestCookieCodec_PointersAndNamedTypes(t *testing.T) {
	n := 42
	if got := roundTrip(t, &n); *got != 42 {
		t.Fatalf("*int: got %d", *got)
	}
	p := &prefs{Theme: "dark", Size: 3}
	if got := roundTrip(t, p); *got != *p {
		t.Fatalf("*struct: got %+v", *got)
	}
	if got := roundTrip(t, *p); got != *p {
		t.Fatalf("struct: got %+v", got)
	}
	if got := roundTrip(t, color("red")); got != "red" {
		t.Fatalf("named string: got %q", got)
	}
	c := color("blue")
	if got := roundTrip(t, &c); *got != "blue" {
		t.Fatalf("*named string: got %q", *got)
	}
	if got := roundTrip(t, true); !got {
		t.Fatal("bool")
	}
	var nilPtr *prefs
	if enc, err := encodeCookieValue(nilPtr); err != nil || enc != "" {
		t.Fatalf("nil pointer: %q %v", enc, err)
	}
}

// A raw ';' ended the cookie value early, so the stored value was silently truncated.
func TestCookieCodec_StringsAreEscaped(t *testing.T) {
	in := "a;b, c=d"
	enc, err := encodeCookieValue(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(enc, "; ,") {
		t.Fatalf("unescaped cookie value %q", enc)
	}
	if got := roundTrip(t, in); got != in {
		t.Fatalf("got %q", got)
	}
	// Values written before escaping existed still decode.
	if got, err := decodeCookieValue[string]("plain-old"); err != nil || got != "plain-old" {
		t.Fatalf("legacy value: %q %v", got, err)
	}
}
