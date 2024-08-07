// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/fluffy-bunny/fluffycore/contracts/GRPCClientFactory (interfaces: IGRPCClientFactory)

// Package GRPCClientFactory is a generated GoMock package.
package GRPCClientFactory

import (
	reflect "reflect"

	grpcclient "github.com/fluffy-bunny/fluffycore/grpcclient"
	gomock "github.com/golang/mock/gomock"
)

// MockIGRPCClientFactory is a mock of IGRPCClientFactory interface.
type MockIGRPCClientFactory struct {
	ctrl     *gomock.Controller
	recorder *MockIGRPCClientFactoryMockRecorder
}

// MockIGRPCClientFactoryMockRecorder is the mock recorder for MockIGRPCClientFactory.
type MockIGRPCClientFactoryMockRecorder struct {
	mock *MockIGRPCClientFactory
}

// NewMockIGRPCClientFactory creates a new mock instance.
func NewMockIGRPCClientFactory(ctrl *gomock.Controller) *MockIGRPCClientFactory {
	mock := &MockIGRPCClientFactory{ctrl: ctrl}
	mock.recorder = &MockIGRPCClientFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIGRPCClientFactory) EXPECT() *MockIGRPCClientFactoryMockRecorder {
	return m.recorder
}

// NewGrpcClient mocks base method.
func (m *MockIGRPCClientFactory) NewGrpcClient(arg0 ...grpcclient.GrpcClientOption) (*grpcclient.GrpcClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "NewGrpcClient", varargs...)
	ret0, _ := ret[0].(*grpcclient.GrpcClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewGrpcClient indicates an expected call of NewGrpcClient.
func (mr *MockIGRPCClientFactoryMockRecorder) NewGrpcClient(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewGrpcClient", reflect.TypeOf((*MockIGRPCClientFactory)(nil).NewGrpcClient), arg0...)
}
