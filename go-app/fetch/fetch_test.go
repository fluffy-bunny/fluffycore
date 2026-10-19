package fetch

import "testing"

// Each call used to write into the package-level DefaultRequestInit, so one call's
// callerID, headers and body leaked into the next.
func TestBuildRequestInit_DoesNotShareState(t *testing.T) {
	a, err := buildRequestInit(&CallInput{Method: "POST", Url: "/a", CallerID: "caller-a",
		CustomHeaders: map[string]string{"X-Csrf-Token": "t"}, Data: map[string]int{"x": 1}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildRequestInit(&CallInput{Url: "/b"})
	if err != nil {
		t.Fatal(err)
	}

	if _, leaked := b["callerID"]; leaked {
		t.Fatal("callerID leaked into a later call")
	}
	if b["method"] != "GET" || b["body"] != nil {
		t.Fatalf("method/body leaked: %v %v", b["method"], b["body"])
	}
	if _, leaked := b["headers"].(map[string]interface{})["X-Csrf-Token"]; leaked {
		t.Fatal("custom header leaked into a later call")
	}
	if a["body"] != `{"x":1}` || a["callerID"] != "caller-a" {
		t.Fatalf("first call's own fields wrong: %v", a)
	}
	for _, k := range []string{"url", "body", "callerID"} {
		if DefaultRequestInit[k] != nil && DefaultRequestInit[k] != "" {
			t.Fatalf("DefaultRequestInit[%q] was modified: %v", k, DefaultRequestInit[k])
		}
	}
}
