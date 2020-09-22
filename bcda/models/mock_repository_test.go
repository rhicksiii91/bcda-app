// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

// Suffixed with _test to avoid placing test code in main path.
// We need this in the same package as models.go to avoid the circular reference.
// Once models.go no longer needs access to Service, we can move this file to a mock package

package models

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// MockRepository is an autogenerated mock type for the Repository type
type MockRepository struct {
	mock.Mock
}

// GetCCLFBeneficiaries provides a mock function with given fields: cclfFileID, ignoredMBIs
func (_m *MockRepository) GetCCLFBeneficiaries(cclfFileID uint, ignoredMBIs []string) ([]*CCLFBeneficiary, error) {
	ret := _m.Called(cclfFileID, ignoredMBIs)

	var r0 []*CCLFBeneficiary
	if rf, ok := ret.Get(0).(func(uint, []string) []*CCLFBeneficiary); ok {
		r0 = rf(cclfFileID, ignoredMBIs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*CCLFBeneficiary)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint, []string) error); ok {
		r1 = rf(cclfFileID, ignoredMBIs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCCLFBeneficiaryMBIs provides a mock function with given fields: cclfFileID
func (_m *MockRepository) GetCCLFBeneficiaryMBIs(cclfFileID uint) ([]string, error) {
	ret := _m.Called(cclfFileID)

	var r0 []string
	if rf, ok := ret.Get(0).(func(uint) []string); ok {
		r0 = rf(cclfFileID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint) error); ok {
		r1 = rf(cclfFileID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLatestCCLFFile provides a mock function with given fields: cmsID, cclfNum, importStatus, lowerBound, upperBound
func (_m *MockRepository) GetLatestCCLFFile(cmsID string, cclfNum int, importStatus string, lowerBound time.Time, upperBound time.Time) (*CCLFFile, error) {
	ret := _m.Called(cmsID, cclfNum, importStatus, lowerBound, upperBound)

	var r0 *CCLFFile
	if rf, ok := ret.Get(0).(func(string, int, string, time.Time, time.Time) *CCLFFile); ok {
		r0 = rf(cmsID, cclfNum, importStatus, lowerBound, upperBound)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*CCLFFile)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int, string, time.Time, time.Time) error); ok {
		r1 = rf(cmsID, cclfNum, importStatus, lowerBound, upperBound)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSuppressedMBIs provides a mock function with given fields: lookbackDays
func (_m *MockRepository) GetSuppressedMBIs(lookbackDays int) ([]string, error) {
	ret := _m.Called(lookbackDays)

	var r0 []string
	if rf, ok := ret.Get(0).(func(int) []string); ok {
		r0 = rf(lookbackDays)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(lookbackDays)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}