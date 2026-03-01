package claimsprincipal

import (
	"testing"

	fluffycore_contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
)

// TestAddClaim_BatchContinues verifies that when one claim in a batch is a
// duplicate or has an empty type, the remaining claims are still processed.
// Previously, `return` was used instead of `continue`, aborting the entire batch.
func TestAddClaim_BatchContinues(t *testing.T) {
	cp := &claimsPrincipal{}
	cp.Ctor()

	// Add initial claim
	cp.AddClaim(fluffycore_contracts_common.Claim{Type: "role", Value: "admin"})

	// Batch: first is duplicate (should skip), second is new (should add)
	cp.AddClaim(
		fluffycore_contracts_common.Claim{Type: "role", Value: "admin"}, // duplicate
		fluffycore_contracts_common.Claim{Type: "role", Value: "user"},  // new
	)

	if !cp.HasClaim(fluffycore_contracts_common.Claim{Type: "role", Value: "user"}) {
		t.Fatal("expected 'role:user' claim to be added after duplicate in batch")
	}
}

// TestAddClaim_EmptyTypeContinues verifies that an empty-type claim in a batch
// doesn't prevent later claims from being added.
func TestAddClaim_EmptyTypeContinues(t *testing.T) {
	cp := &claimsPrincipal{}
	cp.Ctor()

	cp.AddClaim(
		fluffycore_contracts_common.Claim{Type: "", Value: "bad"},    // empty type
		fluffycore_contracts_common.Claim{Type: "role", Value: "ok"}, // should still be added
	)

	if !cp.HasClaim(fluffycore_contracts_common.Claim{Type: "role", Value: "ok"}) {
		t.Fatal("expected 'role:ok' claim to be added after empty-type claim in batch")
	}
}

// TestRemoveClaimType_BatchContinues verifies that removing a non-existent claim type
// doesn't abort removal of subsequent types in the batch.
func TestRemoveClaimType_BatchContinues(t *testing.T) {
	cp := &claimsPrincipal{}
	cp.Ctor()

	cp.AddClaim(
		fluffycore_contracts_common.Claim{Type: "a", Value: "1"},
		fluffycore_contracts_common.Claim{Type: "b", Value: "2"},
	)

	// "nonexistent" is not present, but "b" should still be removed
	cp.RemoveClaimType("nonexistent", "b")

	if cp.HasClaimType("b") {
		t.Fatal("expected claim type 'b' to be removed even though 'nonexistent' was not found")
	}
	if !cp.HasClaimType("a") {
		t.Fatal("expected claim type 'a' to remain")
	}
}

// TestRemoveClaim_BatchContinues verifies that removing a claim not present
// doesn't abort removal of subsequent claims in the batch.
func TestRemoveClaim_BatchContinues(t *testing.T) {
	cp := &claimsPrincipal{}
	cp.Ctor()

	cp.AddClaim(
		fluffycore_contracts_common.Claim{Type: "role", Value: "admin"},
		fluffycore_contracts_common.Claim{Type: "role", Value: "user"},
	)

	// "editor" is not present, but "user" should still be removed
	cp.RemoveClaim(
		fluffycore_contracts_common.Claim{Type: "role", Value: "editor"}, // not present
		fluffycore_contracts_common.Claim{Type: "role", Value: "user"},   // should be removed
	)

	if cp.HasClaim(fluffycore_contracts_common.Claim{Type: "role", Value: "user"}) {
		t.Fatal("expected 'role:user' to be removed even though 'role:editor' was not found")
	}
	if !cp.HasClaim(fluffycore_contracts_common.Claim{Type: "role", Value: "admin"}) {
		t.Fatal("expected 'role:admin' to remain")
	}
}

// TestRemoveClaim_DifferentTypeBatchContinues verifies batch behavior when
// removing a claim whose type does not exist at all.
func TestRemoveClaim_DifferentTypeBatchContinues(t *testing.T) {
	cp := &claimsPrincipal{}
	cp.Ctor()

	cp.AddClaim(
		fluffycore_contracts_common.Claim{Type: "role", Value: "admin"},
	)

	// "scope" type doesn't exist; "role:admin" should still be removed
	cp.RemoveClaim(
		fluffycore_contracts_common.Claim{Type: "scope", Value: "read"}, // type not present
		fluffycore_contracts_common.Claim{Type: "role", Value: "admin"}, // should be removed
	)

	if cp.HasClaim(fluffycore_contracts_common.Claim{Type: "role", Value: "admin"}) {
		t.Fatal("expected 'role:admin' to be removed after missing type in batch")
	}
}
