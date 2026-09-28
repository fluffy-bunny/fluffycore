package localstorage

import "testing"

type color string

type prefs struct {
	Theme string `json:"theme"`
}

func TestLocalStorageCodec_PointersAndNamedTypes(t *testing.T) {
	p := &prefs{Theme: "dark"}
	enc, err := encodeLocalStorageValue(p) // used to panic
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeLocalStorageValue[*prefs](enc)
	if err != nil || got.Theme != "dark" {
		t.Fatalf("*struct: %+v %v", got, err)
	}

	n := uint8(7)
	enc, err = encodeLocalStorageValue(&n)
	if err != nil || enc != "7" {
		t.Fatalf("*uint8: %q %v", enc, err)
	}
	gn, err := decodeLocalStorageValue[*uint8](enc)
	if err != nil || *gn != 7 {
		t.Fatalf("*uint8 decode: %v %v", gn, err)
	}

	gc, err := decodeLocalStorageValue[color]("red") // used to panic
	if err != nil || gc != "red" {
		t.Fatalf("named string: %q %v", gc, err)
	}
	if _, err := decodeLocalStorageValue[uint8]("300"); err == nil {
		t.Fatal("out-of-range uint8 must be an error")
	}
}
