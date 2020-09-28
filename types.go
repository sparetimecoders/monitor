package monitor

import (
	"time"
)

type Checker interface {
	Status() (interface{}, error)
}

type StatusListener interface {
	// CheckFailed is called when a health check state transitions from passing to failing.
	// 	* entry - The recorded state of the health check that triggered the failure
	CheckFailed(entry *State)

	// CheckRecovered is a function that handles the recovery of a failed health check.
	// 	* entry - The recorded state of the health check that triggered the recovery
	// 	* recordedFailures - the total failed health checks that lapsed
	// 	  between the failure and recovery
	//	* failureDurationSeconds - the lapsed time, in seconds, of the recovered failure
	CheckRecovered(entry *State, recordedFailures int64, failureDurationSeconds float64)
	StillFailing(entry *State, recordedFailures int64)
}

// State is a struct that contains the results of the latest
// run of a particular check.
type State struct {
	// Name of the health check
	Name string `json:"name"`

	// Status of the health check state ("ok" or "failed")
	Status string `json:"status"`

	// Err is the error returned from a failed health check
	Err string `json:"error,omitempty"`

	// Details contains more contextual detail about a
	// failing health check.
	Details interface{} `json:"details,omitempty"` // contains JSON message (that can be marshaled)

	// CheckTime is the time of the last health check
	CheckTime time.Time `json:"check_time"`

	ContiguousFailures int64     `json:"num_failures"`     // the number of failures that occurred in a row
	TimeOfFirstFailure time.Time `json:"first_failure_at"` // the time of the initial transitional failure for any given health check
}

// indicates state is failure
func (s *State) isFailure() bool {
	return s.Status == "failed"
}
