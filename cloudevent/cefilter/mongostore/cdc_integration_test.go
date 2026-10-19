package mongostore

// Integration tests against a real replica set (change streams need one). Skipped unless
// CEFILTER_MONGO_URI is set, e.g.:
//
//	docker run -d --name cefilter-cdc-test -p 27199:27017 mongo:8.0.13 \
//	  --replSet rs0 --bind_ip_all --setParameter enableTestCommands=1
//	docker exec cefilter-cdc-test mongosh --eval \
//	  "rs.initiate({_id:'rs0',members:[{_id:0,host:'localhost:27017'}]})"
//	CEFILTER_MONGO_URI='mongodb://localhost:27199/?directConnection=true' go test ./cloudevent/cefilter/mongostore/
//
// enableTestCommands is only needed for the resume-lost test (it uses a failCommand fail
// point); that test skips itself without it.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fluffy-bunny/fluffycore/cloudevent/cefilter"
	"github.com/fluffy-bunny/fluffycore/cloudevent/contracts"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type cdcEnv struct {
	client *mongo.Client
	store  *FilterStore
	coll   *mongo.Collection
	tokens *mongo.Collection
}

func newCDCEnv(t *testing.T) *cdcEnv {
	t.Helper()
	uri := os.Getenv("CEFILTER_MONGO_URI")
	if uri == "" {
		t.Skip("CEFILTER_MONGO_URI not set; skipping change-stream integration test")
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(t.Name())) // db names are limited to 63 chars
	db := client.Database("cefilter_cdc_" + hex.EncodeToString(sum[:8]))
	_ = db.Drop(context.Background())
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})
	coll := db.Collection("cefilters")
	return &cdcEnv{client: client, store: NewFilterStore(coll), coll: coll, tokens: db.Collection("cdc_tokens")}
}

func (e *cdcEnv) upsert(t *testing.T, name, eventType string) {
	t.Helper()
	if err := e.store.Upsert(context.Background(), FilterDocument{Name: name, Hint: name + "-hint", Expr: `type = "` + eventType + `"`}); err != nil {
		t.Fatal(err)
	}
}

func (e *cdcEnv) newCDC() *FilterCDC {
	// Tokens/TokenID set as existing callers do; they must be ignored.
	return &FilterCDC{Filters: e.coll, Tokens: e.tokens, TokenID: "test"}
}

// consumer mirrors the documented wiring: apply every change to an IndexedFilterSet.
type consumer struct {
	t    *testing.T
	ch   chan FilterChange
	gate cefilter.IndexedFilterSet
}

type runHandle struct {
	cancel context.CancelFunc
	done   chan error
}

func (c *consumer) start(cdc *FilterCDC) *runHandle {
	ctx, cancel := context.WithCancel(context.Background())
	h := &runHandle{cancel: cancel, done: make(chan error, 1)}
	go func() { h.done <- cdc.Run(ctx, c.ch) }()
	c.t.Cleanup(cancel)
	return h
}

func (h *runHandle) stop(t *testing.T) {
	t.Helper()
	h.cancel()
	h.wait(t)
}

func (h *runHandle) wait(t *testing.T) error {
	t.Helper()
	select {
	case err := <-h.done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

func (c *consumer) apply(ch FilterChange) {
	c.t.Helper()
	switch ch.Op {
	case ChangeOpUpsert:
		if err := c.gate.Add(contracts.FilterEntry{Name: ch.Doc.Name, Hint: ch.Doc.Hint}, ch.Doc.Expr); err != nil {
			c.t.Fatal(err)
		}
	case ChangeOpDelete:
		c.gate.Remove(ch.Name)
	}
}

// until applies changes until pred returns true, and returns what it saw.
func (c *consumer) until(pred func(FilterChange) bool) []FilterChange {
	c.t.Helper()
	var seen []FilterChange
	timeout := time.After(10 * time.Second)
	for {
		select {
		case ch := <-c.ch:
			seen = append(seen, ch)
			c.apply(ch)
			if pred(ch) {
				return seen
			}
		case <-timeout:
			c.t.Fatalf("timed out; saw %+v", seen)
		}
	}
}

// drain applies whatever a stopped Run delivered before it noticed the cancellation.
func (c *consumer) drain() {
	for {
		select {
		case ch := <-c.ch:
			c.apply(ch)
		default:
			return
		}
	}
}

func isReady(ch FilterChange) bool { return ch.Op == ChangeOpReady }

func opFor(name string, op ChangeOp) func(FilterChange) bool {
	return func(ch FilterChange) bool { return ch.Name == name && ch.Op == op }
}

func (c *consumer) requireMatches(eventType string, wantHints ...string) {
	c.t.Helper()
	got := c.gate.MatchEndpoints(cefilter.Attrs{"type": eventType})
	if strings.Join(got, ",") != strings.Join(wantHints, ",") {
		c.t.Fatalf("type %q: want hints %v, got %v", eventType, wantHints, got)
	}
}

func newConsumer(t *testing.T) *consumer {
	return &consumer{t: t, ch: make(chan FilterChange, 64)}
}

func TestCDC_SnapshotThenLiveChanges(t *testing.T) {
	env := newCDCEnv(t)
	env.upsert(t, "a", "t.a")
	env.upsert(t, "b", "t.b")

	c := newConsumer(t)
	c.start(env.newCDC())
	c.until(isReady)
	c.requireMatches("t.a", "a-hint")
	c.requireMatches("t.b", "b-hint")

	env.upsert(t, "a", "t.a2") // rule changed
	c.until(opFor("a", ChangeOpUpsert))
	c.requireMatches("t.a") // old version must not match any more
	c.requireMatches("t.a2", "a-hint")

	if err := env.store.SoftDelete(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
	c.until(opFor("a", ChangeOpDelete))
	if err := env.store.HardDelete(context.Background(), "b"); err != nil {
		t.Fatal(err)
	}
	c.until(opFor("b", ChangeOpDelete))
	c.requireMatches("t.a2")
	c.requireMatches("t.b")
}

// A new process has an empty filter set. It used to load the previous process's persisted
// resume token, skip the snapshot and emit Ready -- routing nothing.
func TestCDC_NewProcessAlwaysSnapshots(t *testing.T) {
	env := newCDCEnv(t)
	env.upsert(t, "a", "t.a")

	first := newConsumer(t)
	h := first.start(env.newCDC())
	first.until(isReady)
	env.upsert(t, "b", "t.b") // a live event, so a token would have been persisted
	first.until(opFor("b", ChangeOpUpsert))
	h.stop(t)

	restarted := newConsumer(t) // fresh process: new FilterCDC, empty gate
	restarted.start(env.newCDC())
	seen := restarted.until(isReady)
	if len(seen) != 3 {
		t.Fatalf("want snapshot of a, b then Ready, got %+v", seen)
	}
	restarted.requireMatches("t.a", "a-hint")
	restarted.requireMatches("t.b", "b-hint")
}

// A reconnect on the same FilterCDC resumes and replays what happened while it was down.
func TestCDC_ReconnectReplaysChangesMadeWhileDown(t *testing.T) {
	env := newCDCEnv(t)
	env.upsert(t, "a", "t.a")
	env.upsert(t, "b", "t.b")
	env.upsert(t, "c", "t.c")

	cdc := env.newCDC()
	c := newConsumer(t)
	h := c.start(cdc)
	c.until(isReady)
	h.stop(t)
	requireResumable(t, cdc)
	c.drain()

	ctx := context.Background()
	if err := env.store.HardDelete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if err := env.store.SoftDelete(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	env.upsert(t, "c", "t.c2")

	c.start(cdc)
	seen := c.until(func(ch FilterChange) bool { return ch.Op == ChangeOpUpsert && ch.Doc.Expr == `type = "t.c2"` })
	if seen[0].Op != ChangeOpReady {
		t.Fatalf("a reconnect resumes (Ready first, no snapshot), got %+v", seen)
	}
	c.requireMatches("t.a")
	c.requireMatches("t.b")
	c.requireMatches("t.c")
	c.requireMatches("t.c2", "c-hint")
}

// The resume position fell off the oplog: the stream can't replay what was missed, so Run
// re-snapshots -- and must emit Delete for filters removed in the gap, or they keep matching.
func TestCDC_ResumeLost_ResnapshotsAndReconcilesDeletes(t *testing.T) {
	env := newCDCEnv(t)
	env.upsert(t, "a", "t.a")
	env.upsert(t, "b", "t.b")

	cdc := env.newCDC()
	c := newConsumer(t)
	h := c.start(cdc)
	c.until(isReady)
	h.stop(t)
	requireResumable(t, cdc)
	c.drain()

	if err := env.store.HardDelete(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
	env.upsert(t, "d", "t.d")

	// Next change-stream aggregate fails as if the resume point were gone.
	admin := env.client.Database("admin")
	err := admin.RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{{Key: "times", Value: 1}}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{"aggregate"}},
			{Key: "errorCode", Value: 286}, // ChangeStreamHistoryLost
		}},
	}).Err()
	if err != nil {
		t.Skipf("failCommand fail point unavailable (start mongod with --setParameter enableTestCommands=1): %v", err)
	}
	t.Cleanup(func() {
		_ = admin.RunCommand(context.Background(), bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: "off"}}).Err()
	})

	c.start(cdc)
	seen := c.until(isReady)
	var deletedA bool
	for _, ch := range seen {
		deletedA = deletedA || (ch.Op == ChangeOpDelete && ch.Name == "a")
	}
	if !deletedA {
		t.Fatalf("re-snapshot must delete filters removed while the stream was lost, got %+v", seen)
	}
	c.requireMatches("t.a")
	c.requireMatches("t.b", "b-hint")
	c.requireMatches("t.d", "d-hint")

	// And it is tailing again.
	env.upsert(t, "e", "t.e")
	c.until(opFor("e", ChangeOpUpsert))
}

// Dropping the collection invalidates the stream; the next Run re-snapshots (now empty) and
// deletes everything it had reported.
func TestCDC_InvalidatedStream_NextRunReconciles(t *testing.T) {
	env := newCDCEnv(t)
	env.upsert(t, "a", "t.a")

	cdc := env.newCDC()
	c := newConsumer(t)
	h := c.start(cdc)
	c.until(isReady)

	if err := env.coll.Drop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := h.wait(t); err == nil || !strings.Contains(err.Error(), "invalidated") {
		t.Fatalf("want an invalidated error, got %v", err)
	}

	c.start(cdc)
	c.until(isReady)
	c.requireMatches("t.a")
}

func TestCDC_ConcurrentRunRejected(t *testing.T) {
	env := newCDCEnv(t)
	cdc := env.newCDC()
	c := newConsumer(t)
	c.start(cdc)
	c.until(isReady)
	if err := cdc.Run(context.Background(), make(chan FilterChange)); err == nil {
		t.Fatal("a second concurrent Run must be rejected")
	}
}

// requireResumable asserts the next Run will resume rather than snapshot, so the test
// exercises the resume path it claims to.
func requireResumable(t *testing.T, cdc *FilterCDC) {
	t.Helper()
	if cdc.resumeToken == nil && cdc.resumeTime == nil {
		t.Fatal("FilterCDC holds no resume position after a clean stop")
	}
}

// Stopped after the snapshot but before the stream produced a resume token (here: blocked
// delivering Ready). The next Run must resume from the snapshot's pinned time, not snapshot
// again.
func TestCDC_StopBeforeFirstToken_ResumesFromSnapshotTime(t *testing.T) {
	env := newCDCEnv(t)
	env.upsert(t, "a", "t.a")

	cdc := env.newCDC()
	c := &consumer{t: t, ch: make(chan FilterChange, 1)} // room for the snapshot only
	h := c.start(cdc)
	deadline := time.Now().Add(10 * time.Second)
	for len(c.ch) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond) // let Run open the stream and block on Ready
	h.stop(t)
	requireResumable(t, cdc)
	c.drain()

	c.start(cdc)
	if seen := c.until(isReady); len(seen) != 1 {
		t.Fatalf("want a resume (Ready only), got a re-snapshot: %+v", seen)
	}
}
