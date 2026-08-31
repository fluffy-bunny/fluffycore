package auth

import (
	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	proto_helloworld "github.com/fluffy-bunny/fluffycore/proto/helloworld"
	services_common_claimsprincipal "github.com/fluffy-bunny/fluffycore/services/common/claimsprincipal"
	fluffycore_wellknown "github.com/fluffy-bunny/fluffycore/wellknown"
)

var writeEndpoints = []string{
	proto_helloworld.Greeter_SayHelloAuth_FullMethodName,
}
var noAuthEndpoints = []string{
	"/grpc.health.v1.Health/Check",
	proto_helloworld.Greeter_SayHello_FullMethodName,
	proto_helloworld.Greeter_SayHelloDownstream_FullMethodName,
}

// secretEndpoints may be called with EITHER a normal JWT bearer token OR
// mutual TLS (a verified client certificate) -- neither is required if the
// other is present. See jwtAuthenticated/mtlsVerified below.
var secretEndpoints = []string{
	proto_helloworld.Greeter_SetSecret_FullMethodName,
	proto_helloworld.Greeter_GetSecret_FullMethodName,
}

// jwtAuthenticated matches a request that carries the "sub" claim type AND
// its value is not AnonymousSubject. "sub" alone isn't enough: the JWT
// middleware also adds sub=anonymous when there's no Authorization header at
// all (e.g. an mTLS-only caller with no bearer token), so excluding that
// specific value is what actually distinguishes "presented and validated a
// real JWT" from "presented nothing".
var jwtAuthenticated = &services_common_claimsprincipal.ClaimsAST{
	ClaimFacts: []contracts_common.IClaimFact{
		services_common_claimsprincipal.NewClaimFactType(fluffycore_wellknown.ClaimTypeSub),
	},
	Not: []contracts_common.IClaimsValidator{
		&services_common_claimsprincipal.ClaimsAST{
			ClaimFacts: []contracts_common.IClaimFact{
				services_common_claimsprincipal.NewClaimFact(contracts_common.Claim{
					Type:  fluffycore_wellknown.ClaimTypeSub,
					Value: fluffycore_wellknown.AnonymousSubject,
				}),
			},
		},
	},
}

// mtlsVerified matches a request whose connection presented a client
// certificate that verified against the server's configured client CA bundle
// -- see middleware/auth/mtls, which populates this claim.
var mtlsVerified = &services_common_claimsprincipal.ClaimsAST{
	ClaimFacts: []contracts_common.IClaimFact{
		services_common_claimsprincipal.NewClaimFact(contracts_common.Claim{
			Type:  fluffycore_wellknown.ClaimTypeMTLSVerified,
			Value: "true",
		}),
	},
}

func BuildGrpcEntrypointPermissionsClaimsMap() map[string]contracts_common.IEntryPointConfig {
	entryPointClaimsBuilder := services_common_claimsprincipal.NewEntryPointClaimsBuilder()
	for _, endpoint := range noAuthEndpoints {
		entryPointClaimsBuilder.WithGrpcEntrypointPermissionsClaimsMapOpen(endpoint)
	}
	for _, endpoint := range writeEndpoints {
		entrypointConfig := &services_common_claimsprincipal.EntryPointConfig{
			FullMethodName: endpoint,
			ClaimsAST: &services_common_claimsprincipal.ClaimsAST{
				Or: []contracts_common.IClaimsValidator{
					&services_common_claimsprincipal.ClaimsAST{
						ClaimFacts: []contracts_common.IClaimFact{
							services_common_claimsprincipal.NewClaimFact(contracts_common.Claim{
								Type:  "permissions",
								Value: "write",
							}),
						},
					},
				},
			},
		}
		entryPointClaimsBuilder.EntrypointClaimsMap[endpoint] = entrypointConfig
	}
	for _, endpoint := range secretEndpoints {
		entrypointConfig := &services_common_claimsprincipal.EntryPointConfig{
			FullMethodName: endpoint,
			// A single entry in the root's Or -- that's what makes ClaimsAST
			// evaluate this node's own children with OR semantics (Validate()
			// always starts the root in AND mode; a lone Or-child is what flips
			// the operand). jwtAuthenticated/mtlsVerified are then peers of
			// *that* node's And slice -- which, precisely because the node
			// itself is being evaluated with OR, combines its And-peers with
			// "any one true wins" too. jwtAuthenticated specifically must sit in
			// an And slot (not Or) so it recurses internally with AND semantics
			// -- it needs sub-claim-present AND NOT sub=anonymous, both true,
			// not either. See claims_ast_test.go's TestClaimsAndOrGroup for the
			// same "single Or-entry wrapping an And-list" idiom.
			ClaimsAST: &services_common_claimsprincipal.ClaimsAST{
				Or: []contracts_common.IClaimsValidator{
					&services_common_claimsprincipal.ClaimsAST{
						And: []contracts_common.IClaimsValidator{
							jwtAuthenticated,
							mtlsVerified,
						},
					},
				},
			},
		}
		entryPointClaimsBuilder.EntrypointClaimsMap[endpoint] = entrypointConfig
	}
	entryPointClaimsBuilder.DumpExpressions()
	return entryPointClaimsBuilder.GetEntryPointClaimsMap()
}
