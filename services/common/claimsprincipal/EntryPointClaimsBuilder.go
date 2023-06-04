package claimsprincipal

import (
	"fmt"

	contracts_claimfact "github.com/fluffy-bunny/fluffycore/contracts/common"
	"github.com/fluffy-bunny/fluffycore/utils"
)

// EntryPointClaimsBuilder struct
type EntryPointClaimsBuilder struct {
	EntrypointClaimsMap map[string]*EntryPointConfig
}

// NewEntryPointClaimsBuilder ...
func NewEntryPointClaimsBuilder() *EntryPointClaimsBuilder {
	return &EntryPointClaimsBuilder{
		EntrypointClaimsMap: make(map[string]*EntryPointConfig),
	}
}
func (s *EntryPointClaimsBuilder) GetEntryPointClaimsMap() map[string]*EntryPointConfig {
	return s.EntrypointClaimsMap
}

// WithGrpcEntrypointPermissionsClaimsMapOpen helper to add a single entrypoint config
func (s *EntryPointClaimsBuilder) WithGrpcEntrypointPermissionsClaimsMapOpen(fullMethodName string) *EntryPointClaimsBuilder {
	s.ensureEntry(fullMethodName)
	return s
}

// WithGrpcEntrypointClams helper to add a single entrypoint config
func (s *EntryPointClaimsBuilder) WithGrpcEntrypointClams(fullMethodName string, claims ...contracts_claimfact.IClaimFact) *EntryPointClaimsBuilder {
	ast := s.GetClaimsAST(fullMethodName)
	ast.ClaimFacts = append(ast.ClaimFacts, claims...)
	return s
}

// GetClaimsAST ...
func (s *EntryPointClaimsBuilder) GetClaimsAST(fullMethodName string) *ClaimsAST {
	result := s.ensureEntry(fullMethodName)
	return result.ClaimsAST
}

func (s *EntryPointClaimsBuilder) ensureEntry(fullMethodName string) *EntryPointConfig {
	result, ok := s.EntrypointClaimsMap[fullMethodName]
	if !ok {
		result = &EntryPointConfig{
			FullMethodName: fullMethodName,
			ClaimsAST:      &ClaimsAST{},
		}
		s.EntrypointClaimsMap[fullMethodName] = result
	}
	return result
}

// DumpExpressions ...
func (s *EntryPointClaimsBuilder) DumpExpressions() {

	type entrypoint struct {
		FullMethodName string
		Expression     string
	}
	type fullExpressions struct {
		EntryPoints []entrypoint
	}
	fullE := fullExpressions{}
	fmt.Println("")
	fmt.Println("EntryPointClaimsBuilder auth config:")
	fmt.Println("==================================================================")
	for _, entry := range s.EntrypointClaimsMap {
		fullE.EntryPoints = append(fullE.EntryPoints, entrypoint{
			FullMethodName: entry.FullMethodName,
			Expression:     entry.ClaimsAST.String(),
		})
	}
	fmt.Println(utils.PrettyJSON(fullE))

}
