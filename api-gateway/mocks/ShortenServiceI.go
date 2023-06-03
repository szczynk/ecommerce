// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	model "api-gateway-go/model"

	mock "github.com/stretchr/testify/mock"
)

// ShortenServiceI is an autogenerated mock type for the ShortenServiceI type
type ShortenServiceI struct {
	mock.Mock
}

// Create provides a mock function with given fields: shortenReq
func (_m *ShortenServiceI) Create(shortenReq *model.ShortenReq) (*model.APIManagement, error) {
	ret := _m.Called(shortenReq)

	var r0 *model.APIManagement
	var r1 error
	if rf, ok := ret.Get(0).(func(*model.ShortenReq) (*model.APIManagement, error)); ok {
		return rf(shortenReq)
	}
	if rf, ok := ret.Get(0).(func(*model.ShortenReq) *model.APIManagement); ok {
		r0 = rf(shortenReq)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.APIManagement)
		}
	}

	if rf, ok := ret.Get(1).(func(*model.ShortenReq) error); ok {
		r1 = rf(shortenReq)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Get provides a mock function with given fields: hashedURL
func (_m *ShortenServiceI) Get(hashedURL string) (*model.APIManagement, error) {
	ret := _m.Called(hashedURL)

	var r0 *model.APIManagement
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*model.APIManagement, error)); ok {
		return rf(hashedURL)
	}
	if rf, ok := ret.Get(0).(func(string) *model.APIManagement); ok {
		r0 = rf(hashedURL)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*model.APIManagement)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(hashedURL)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewShortenServiceI interface {
	mock.TestingT
	Cleanup(func())
}

// NewShortenServiceI creates a new instance of ShortenServiceI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewShortenServiceI(t mockConstructorTestingTNewShortenServiceI) *ShortenServiceI {
	mock := &ShortenServiceI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
