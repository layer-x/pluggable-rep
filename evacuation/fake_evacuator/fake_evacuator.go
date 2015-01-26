// This file was generated by counterfeiter
package fake_evacuator

import (
	"sync"

	"github.com/cloudfoundry-incubator/rep/evacuation"
)

type FakeEvacuationContext struct {
	EvacuatingStub        func() bool
	evacuatingMutex       sync.RWMutex
	evacuatingArgsForCall []struct{}
	evacuatingReturns struct {
		result1 bool
	}
}

func (fake *FakeEvacuationContext) Evacuating() bool {
	fake.evacuatingMutex.Lock()
	fake.evacuatingArgsForCall = append(fake.evacuatingArgsForCall, struct{}{})
	fake.evacuatingMutex.Unlock()
	if fake.EvacuatingStub != nil {
		return fake.EvacuatingStub()
	} else {
		return fake.evacuatingReturns.result1
	}
}

func (fake *FakeEvacuationContext) EvacuatingCallCount() int {
	fake.evacuatingMutex.RLock()
	defer fake.evacuatingMutex.RUnlock()
	return len(fake.evacuatingArgsForCall)
}

func (fake *FakeEvacuationContext) EvacuatingReturns(result1 bool) {
	fake.EvacuatingStub = nil
	fake.evacuatingReturns = struct {
		result1 bool
	}{result1}
}

var _ evacuation.EvacuationContext = new(FakeEvacuationContext)
