package claimsprincipal

import (
	"fmt"

	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
	utils "github.com/fluffy-bunny/fluffycore/utils"
)

// EntryPointClaimsBuilder struct
type EntryPointClaimsBuilder struct {
	EntrypointClaimsMap map[string]contracts_common.IEntryPointConfig
}

// NewEntryPointClaimsBuilder ...
func NewEntryPointClaimsBuilder() *EntryPointClaimsBuilder {
	return &EntryPointClaimsBuilder{
		EntrypointClaimsMap: make(map[string]contracts_common.IEntryPointConfig),
	}
}
func (s *EntryPointClaimsBuilder) GetEntryPointClaimsMap() map[string]contracts_common.IEntryPointConfig {
	return s.EntrypointClaimsMap
}

// WithGrpcEntrypointPermissionsClaimsMapOpen helper to add a single entrypoint config
func (s *EntryPointClaimsBuilder) WithGrpcEntrypointPermissionsClaimsMapOpen(fullMethodName string) *EntryPointClaimsBuilder {
	s.ensureEntry(fullMethodName)
	return s
}

// WithGrpcEntrypointClams helper to add a single entrypoint config
func (s *EntryPointClaimsBuilder) WithGrpcEntrypointClams(fullMethodName string, claims ...contracts_common.IClaimFact) *EntryPointClaimsBuilder {
	ast := s.GetClaimsAST(fullMethodName)
	ast.AppendClaimsFact(claims...)
	return s
}

// GetClaimsAST ...
func (s *EntryPointClaimsBuilder) GetClaimsAST(fullMethodName string) contracts_common.IClaimsAST {
	result := s.ensureEntry(fullMethodName)
	return result.GetClaimsAST()
}

func (s *EntryPointClaimsBuilder) ensureEntry(fullMethodName string) contracts_common.IEntryPointConfig {
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
			FullMethodName: entry.GetFullMethodName(),
			Expression:     entry.GetExpression(),
		})
	}
	fmt.Println(utils.PrettyJSON(fullE))

}
