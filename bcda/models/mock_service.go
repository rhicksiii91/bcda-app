// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package models

import (
	que "github.com/bgentry/que-go"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// MockService is an autogenerated mock type for the Service type
type MockService struct {
	mock.Mock
}

// GetQueJobs provides a mock function with given fields: cmsID, job, resourceTypes, since, reqType
func (_m *MockService) GetQueJobs(cmsID string, job *Job, resourceTypes []string, since time.Time, reqType RequestType) ([]*que.Job, error) {
	ret := _m.Called(cmsID, job, resourceTypes, since, reqType)

	var r0 []*que.Job
	if rf, ok := ret.Get(0).(func(string, *Job, []string, time.Time, RequestType) []*que.Job); ok {
		r0 = rf(cmsID, job, resourceTypes, since, reqType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*que.Job)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, *Job, []string, time.Time, RequestType) error); ok {
		r1 = rf(cmsID, job, resourceTypes, since, reqType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}