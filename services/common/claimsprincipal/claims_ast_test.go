package claimsprincipal

import (
	"fmt"
	"testing"

	contracts_auth "github.com/fluffy-bunny/fluffycore/contracts/common"
	contracts_claimfact "github.com/fluffy-bunny/fluffycore/contracts/common"
	contracts_claimsprincipal "github.com/fluffy-bunny/fluffycore/contracts/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimsEmpty(t *testing.T) {
	// ()
	perms := ClaimsAST{}

	assert.Equal(t, "()", perms.String())
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D")))
}

func TestClaimsRootOnly(t *testing.T) {
	// (A && B)
	perms := ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "A",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "B",
			}),
		},
	}

	fmt.Println(perms.String())
	assert.Equal(t, "(has_claim(permissions|A) AND has_claim(permissions|B))", perms.String())
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("B", "C", "D")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "C", "D")))
}
func TestClaimsRootOnlyTypeOnly(t *testing.T) {
	// (A && B)
	perms := ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFactType("permissions"),
		},
	}
	fmt.Println(perms.String())
	require.Equal(t, "(has_claim_type(permissions))", perms.String())
	require.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D")))
}
func TestClaimsAndOrGroup(t *testing.T) {
	// Complex
	// if you have the All claim, we are done
	// OR you MUST have the org claim (don't care about the value) AND you must have the permissions claim to match your org.
	perms := ClaimsAST{

		Or: []contracts_auth.IClaimsValidator{
			&ClaimsAST{
				ClaimFacts: []contracts_claimfact.IClaimFact{
					NewClaimFactType("all"),
				},
				And: []contracts_auth.IClaimsValidator{
					&ClaimsAST{
						ClaimFacts: []contracts_claimfact.IClaimFact{
							NewClaimFactType("org"),
						},
						And: []contracts_auth.IClaimsValidator{
							&ClaimsAST{

								Or: []contracts_auth.IClaimsValidator{
									&ClaimsAST{
										ClaimFacts: []contracts_claimfact.IClaimFact{
											NewClaimFact(contracts_claimsprincipal.Claim{
												Type:  "permissions",
												Value: "A",
											}),
											NewClaimFact(contracts_claimsprincipal.Claim{
												Type:  "permissions",
												Value: "B",
											}),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	fmt.Println(perms.String())

	require.Equal(t, "((has_claim_type(all) OR (has_claim_type(org) AND ((has_claim(permissions|A) OR has_claim(permissions|B))))))", perms.String())

	cp := NewmockClaimsPrincipalToken("secret")
	cp.AddClaim(contracts_claimsprincipal.Claim{
		Type:  "all",
		Value: "true",
	})
	require.True(t, perms.Validate(cp))

	cp = NewmockClaimsPrincipalToken("A")
	cp.AddClaim(contracts_claimsprincipal.Claim{
		Type:  "org",
		Value: "org1234",
	})
	require.True(t, perms.Validate(cp))

	cp = NewmockClaimsPrincipalToken("B")
	cp.AddClaim(contracts_claimsprincipal.Claim{
		Type:  "org",
		Value: "org1234",
	})
	require.True(t, perms.Validate(cp))

	cp = NewmockClaimsPrincipalToken("C")
	cp.AddClaim(contracts_claimsprincipal.Claim{
		Type:  "org",
		Value: "org1234",
	})
	require.False(t, perms.Validate(cp))
}
func TestClaimsRootOnlyTypeOnlyFail(t *testing.T) {
	// (A && B)
	perms := ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFactType("secret"),
		},
	}

	require.Equal(t, "(has_claim_type(secret))", perms.String())
	require.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D")))
}
func TestClaimsBranchAnd(t *testing.T) {
	// ((A && B))
	var ands []contracts_auth.IClaimsValidator
	ands = append(ands, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "A",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "B",
			}),
		},
	})
	perms := ClaimsAST{
		And: ands,
	}

	assert.Equal(t, "((has_claim(permissions|A) AND has_claim(permissions|B)))", perms.String())
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("B", "C", "D")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "C", "D")))

	_ = perms
}

func TestClaimsBranchOr(t *testing.T) {
	// ((A || B))
	var ors []contracts_auth.IClaimsValidator
	ors = append(ors, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "A",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "B",
			}),
		},
	})
	perms := ClaimsAST{
		Or: ors,
	}

	require.Equal(t, "((has_claim(permissions|A) OR has_claim(permissions|B)))", perms.String())
	require.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D")))
	require.True(t, perms.Validate(NewmockClaimsPrincipalToken("B", "C", "D")))
	require.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "C", "D")))
	require.False(t, perms.Validate(NewmockClaimsPrincipalToken("X", "C", "D")))
	require.False(t, perms.Validate(NewmockClaimsPrincipalToken("C", "D")))
}
func TestClaimsBranchOrTypeOnly(t *testing.T) {
	// ((A || B))
	var ors []contracts_auth.IClaimsValidator
	ors = append(ors, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFactType("permissions"),
		},
	})
	perms := ClaimsAST{
		Or: ors,
	}

	require.Equal(t, "((has_claim_type(permissions)))", perms.String())
	require.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D")))
}
func TestClaimsBranchNot(t *testing.T) {
	// Not inherits the operand from it's parent, and in this case
	// (!(A && B))
	var nots []contracts_auth.IClaimsValidator
	nots = append(nots, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "A",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "B",
			}),
		},
	})
	perms := ClaimsAST{
		Not: nots,
	}

	assert.Equal(t, "(!(has_claim(permissions|A) AND has_claim(permissions|B)))", perms.String())
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("C", "D", "E", "F")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "C", "D")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("B", "C", "D")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B")))
}

func TestClaimsBranchNotNested(t *testing.T) {
	// (!((A || B)))
	var ors []contracts_auth.IClaimsValidator
	ors = append(ors, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "A",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "B",
			}),
		},
	})

	var nots []contracts_auth.IClaimsValidator
	nots = append(nots, &ClaimsAST{
		Or: ors,
	})
	perms := ClaimsAST{
		Not: nots,
	}

	assert.Equal(t, "(!((has_claim(permissions|A) OR has_claim(permissions|B))))", perms.String())
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("C", "D", "E", "F")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "C", "D")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("B", "C", "D")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B")))
}

func TestClaimsDocSample(t *testing.T) {
	// (A && B && (C || D) && (E || F || (G && H)) && !((I && J)))
	var ands1 []contracts_auth.IClaimsValidator
	ands1 = append(ands1, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "G",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "H",
			}),
		},
	})
	var ors []contracts_auth.IClaimsValidator
	ors = append(ors, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "C",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "D",
			}),
		},
	})
	ors = append(ors, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "E",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "F",
			}),
		},
		And: ands1,
	})

	var ands2 []contracts_auth.IClaimsValidator
	ands2 = append(ands2, &ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "I",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "J",
			}),
		},
	})
	var nots []contracts_auth.IClaimsValidator
	nots = append(nots, &ClaimsAST{
		And: ands2,
	})
	perms := ClaimsAST{
		ClaimFacts: []contracts_claimfact.IClaimFact{
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "A",
			}),
			NewClaimFact(contracts_claimsprincipal.Claim{
				Type:  "permissions",
				Value: "B",
			}),
		},
		Or:  ors,
		Not: nots,
	}
	/*
		perms := Claims{
			Values: []string{"A", "B"},
			Or: []*Claims{
				{Values: []string{"C", "D"}},
				{
					Values: []string{"E", "F"},
					And: []*Claims{
						{Values: []string{"G", "H"}},
					},
				},
			},
			Not: []*Claims{
				{
					And: []*Claims{
						{Values: []string{"I", "J"}},
					},
				},
			},
		}
	*/
	fmt.Println(perms.String())
	assert.Equal(t, "(has_claim(permissions|A) AND has_claim(permissions|B) AND (has_claim(permissions|C) OR has_claim(permissions|D)) AND (has_claim(permissions|E) OR has_claim(permissions|F) OR (has_claim(permissions|G) AND has_claim(permissions|H))) AND !((has_claim(permissions|I) AND has_claim(permissions|J))))", perms.String())

	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "E", "F", "G", "H")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "E", "F", "G", "H", "I")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "E", "F", "G", "H", "J")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "E", "G", "H")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "E")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "D", "F")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "G", "H")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "F", "G", "H", "I")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "E", "G", "H")))
	assert.True(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "D", "F", "G", "H", "J")))

	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("B", "C", "D", "E", "F", "G", "H")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "C", "D", "E", "F", "G", "H")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "E", "F", "G", "H", "I")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "H", "J")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "H")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "G")))
	assert.False(t, perms.Validate(NewmockClaimsPrincipalToken("A", "B", "C", "D", "F", "G", "H", "I", "J")))
}
func NewmockClaimsPrincipalToken(perms ...string) contracts_claimsprincipal.IClaimsPrincipal {

	cp := NewIClaimsPrincipal()

	for _, perm := range perms {
		claim := contracts_claimsprincipal.Claim{
			Type:  "permissions",
			Value: perm,
		}
		cp.AddClaim(claim)
	}
	return cp
}
