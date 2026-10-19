package mongostore

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestIsResumeLost(t *testing.T) {
	for _, code := range []int32{136, 260, 280, 286} {
		err := fmt.Errorf("cefilter cdc: watch: %w", mongo.CommandError{Code: code})
		if !isResumeLost(err) {
			t.Errorf("code %d should mean the resume position is lost", code)
		}
	}
	for _, err := range []error{nil, errors.New("network"), mongo.CommandError{Code: 6 /* HostUnreachable */}} {
		if isResumeLost(err) {
			t.Errorf("%v must not trigger a re-snapshot", err)
		}
	}
}

func TestStaleNames(t *testing.T) {
	set := func(names ...string) map[string]struct{} {
		m := map[string]struct{}{}
		for _, n := range names {
			m[n] = struct{}{}
		}
		return m
	}
	got := staleNames(set("a", "b", "c"), set("b", "d"))
	sort.Strings(got)
	if fmt.Sprint(got) != "[a c]" {
		t.Fatalf("got %v", got)
	}
	if len(staleNames(nil, set("a"))) != 0 {
		t.Fatal("nothing reported yet means nothing stale")
	}
}

func TestTrack(t *testing.T) {
	c := &FilterCDC{}
	c.track(FilterChange{Op: ChangeOpUpsert, Name: "a"})
	c.track(FilterChange{Op: ChangeOpUpsert, Name: "b"})
	c.track(FilterChange{Op: ChangeOpDelete, Name: "a"})
	c.track(FilterChange{Op: ChangeOpReady})
	if len(c.live) != 1 {
		t.Fatalf("want only b live, got %v", c.live)
	}
	if _, ok := c.live["b"]; !ok {
		t.Fatalf("want b live, got %v", c.live)
	}
}
