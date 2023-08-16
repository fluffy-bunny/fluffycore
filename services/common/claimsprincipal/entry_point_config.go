package claimsprincipal

import (
	contracts_common "github.com/fluffy-bunny/fluffycore/contracts/common"
)

type EntryPointConfig struct {
	FullMethodName string     `mapstructure:"FULL_METHOD_NAME"`
	ClaimsAST      *ClaimsAST `mapstructure:"CLAIMS_CONFIG"`
}

func (s *EntryPointConfig) GetFullMethodName() string {
	return s.FullMethodName
}
func (s *EntryPointConfig) GetClaimsAST() contracts_common.IClaimsAST {
	return s.ClaimsAST
}
func (s *EntryPointConfig) GetExpression() string {
	return s.ClaimsAST.String()
}
