package monitor

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_AlreadyStarted(t *testing.T) {
	mon := New()

	require.NoError(t, mon.Start())
	defer mon.Stop()
	require.Error(t, mon.Start())
}

func Test_AddSameConfigReturnsError(t *testing.T) {
	mon := New()

	require.NoError(t, mon.AddCheck(&Config{
		Name:       "test",
		Checker:    &mockChecker{},
		Interval:   time.Second,
		OnComplete: nil,
	}))

	require.Error(t, mon.AddCheck(&Config{
		Name:       "test",
		Checker:    &mockChecker{},
		Interval:   time.Second,
		OnComplete: nil,
	}))
}

func Test_StopCleanUp(t *testing.T) {
	mon := New()

	require.NoError(t, mon.AddCheck(&Config{
		Name:       "test",
		Checker:    &mockChecker{},
		Interval:   100 * time.Millisecond,
		OnComplete: nil,
	}))
	require.NoError(t, mon.Start())
	time.Sleep(time.Second * 2)
	stateBeforeStop, _ := mon.State()
	require.GreaterOrEqual(t, len(stateBeforeStop), 1)
	require.NoError(t, mon.Stop())
	state, err := mon.State()
	require.NoError(t, err)
	require.Equal(t, 0, len(state))
}

func Test_HandlerGetsCalled(t *testing.T) {
	mon := New()

	handler := onCompleteHandler{}
	require.NoError(t, mon.AddCheck(&Config{
		Name:       "test",
		Checker:    &mockChecker{failAfter: 1, recoverAfter: 300},
		Interval:   100 * time.Millisecond,
		OnComplete: handler.onComplete,
	}))
	require.NoError(t, mon.Start())
	defer mon.Stop()
	time.Sleep(time.Second * 1)
	require.GreaterOrEqual(t, len(handler.states), 1)
	require.Equal(t, "ok", handler.states[0].Status)
	require.Equal(t, "failed", handler.states[1].Status)
}

func Test_StatusListenerCalled(t *testing.T) {
	mon := New()

	require.NoError(t, mon.AddCheck(&Config{
		Name:       "test",
		Checker:    &mockChecker{failAfter: 0, recoverAfter: 5},
		Interval:   100 * time.Millisecond,
		OnComplete: nil,
	}))
	listener := mockStatusListener{}
	mon.StatusListener = &listener
	require.NoError(t, mon.Start())
	defer mon.Stop()
	time.Sleep(time.Second * 2)
	require.Equal(t, 1, listener.checkFailedCalled)
	require.Equal(t, 3, listener.stillFailingCalled)
	require.Equal(t, 1, listener.checkRecoveredCalled)
}

func Test_StopIndividualCheck(t *testing.T) {
	mon := New()

	require.NoError(t, mon.AddCheck(&Config{
		Name:       "test",
		Checker:    &mockChecker{failAfter: 0, recoverAfter: 5},
		Interval:   100 * time.Millisecond,
		OnComplete: nil,
	}))

	require.NoError(t, mon.AddCheck(&Config{
		Name:       "test_to_stop",
		Checker:    &mockChecker{failAfter: 0, recoverAfter: 5},
		Interval:   100 * time.Millisecond,
		OnComplete: nil,
	}))
	require.NoError(t, mon.Start())
	defer mon.Stop()
	time.Sleep(time.Millisecond * 500)
	mon.StopCheck("test_to_stop")
	time.Sleep(time.Millisecond * 500)
	require.Equal(t, 1, len(mon.safeGetStates()))
}

func Test_StopNonExistingCheckShouldReturnError(t *testing.T) {
	mon := New()
	require.Error(t, mon.StopCheck("test_to_stop"))
}

type mockStatusListener struct {
	checkFailedCalled    int
	checkRecoveredCalled int
	stillFailingCalled   int
}

func (m *mockStatusListener) CheckFailed(entry *State) {
	m.checkFailedCalled = m.checkFailedCalled + 1
}

func (m *mockStatusListener) CheckRecovered(entry *State, recordedFailures int64, failureDurationSeconds float64) {
	m.checkRecoveredCalled = m.checkRecoveredCalled + 1
}

func (m *mockStatusListener) StillFailing(entry *State, recordedFailures int64) {
	m.stillFailingCalled = m.stillFailingCalled + 1
}

type onCompleteHandler struct {
	states []*State
}

func (h *onCompleteHandler) onComplete(state *State) {
	h.states = append(h.states, state)
}

type mockChecker struct {
	failAfter    int
	checks       int
	recoverAfter int
}

func (mc *mockChecker) Status() (interface{}, error) {
	mc.checks = mc.checks + 1
	if mc.checks <= mc.failAfter || mc.checks >= mc.recoverAfter {
		return "Working just fine", nil
	} else {
		return "", fmt.Errorf("failure")
	}
}
