package monitor

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Monitor contains internal go-health internal structures.
type Monitor struct {
	// StatusListener will report failures and recoveries
	StatusListener StatusListener
	// RandomStartTimeMillis returns a random delay to wait before starting the checks (one for each check)
	RandomStartTimeMillis func() int

	configs     []*Config
	states      map[string]State
	statesLock  sync.Mutex
	runnersLock sync.Mutex
	runners     map[string]chan struct{} // contains map of active runners w/ a stop channel
	started     bool
}

// New returns a new instance of the Monitor struct.
func New() *Monitor {
	return &Monitor{
		configs:     make([]*Config, 0),
		states:      make(map[string]State, 0),
		runners:     make(map[string]chan struct{}, 0),
		statesLock:  sync.Mutex{},
		runnersLock: sync.Mutex{},
		RandomStartTimeMillis: func() int {
			return 0
		},
	}
}

// AddCheck is used for adding a single check definition to the current health instance.
func (h *Monitor) AddCheck(cfg ...*Config) error {
	for _, existing := range h.configs {
		for _, c := range cfg {
			if c.Name == existing.Name {
				return fmt.Errorf("config with name %s already exists", c.Name)
			}
		}
	}
	h.configs = append(h.configs, cfg...)
	return nil
}

func (h *Monitor) RemoveCheck(cfg *Config) error {
	for idx, existing := range h.configs {
		if cfg.Name == existing.Name {
			if h.started {
				if err := h.StopCheck(cfg.Name); err != nil {
					return err
				}
				fmt.Printf("stopped check %s\n", cfg.Name)
			}
			h.configs = append(h.configs[:idx], h.configs[idx+1:]...)
			fmt.Printf("removed check %s\n", cfg.Name)
			return nil
		}
	}
	return fmt.Errorf("no check found with name %s", cfg.Name)
}

// Start will start all of the defined health checks. Each of the checks run in
// their own goroutines (as "time.Ticker").
func (h *Monitor) Start() error {
	if h.started {
		return errors.New("monitor already started")
	}
	h.started = true
	for _, c := range h.configs {
		h.startRunnerForConfig(c)
	}

	return nil
}

func (h *Monitor) startRunnerForConfig(c *Config) {
	stop := make(chan struct{})
	h.startRunner(c, stop)
	h.runnersLock.Lock()
	defer h.runnersLock.Unlock()
	h.runners[c.Name] = stop
	fmt.Printf("started check %s\n", c.Name)

}

func (h *Monitor) StopCheck(name string) error {
	h.runnersLock.Lock()
	defer h.runnersLock.Unlock()

	if stop := h.runners[name]; stop != nil {
		fmt.Printf("stopping check %s\n", name)
		close(stop)
		delete(h.runners, name)
	} else {
		return fmt.Errorf("failed to find check with name %s", name)
	}

	// Reset state
	h.statesLock.Lock()
	defer h.statesLock.Unlock()
	delete(h.states, name)
	return nil
}

func (h *Monitor) StartCheck(name string) error {
	var found *Config
	for _, existing := range h.configs {
		if name == existing.Name {
			found = existing
		}
	}
	if found == nil {
		return fmt.Errorf("failed to find check with name %s", name)
	}
	if stop := h.runners[name]; stop != nil {
		return fmt.Errorf("check already running")
	} else {
		h.startRunnerForConfig(found)
	}
	return nil
}

// Stop will cause all of the running health checks to be stopped. Additionally,
// all existing check states will be reset.
func (h *Monitor) Stop() error {
	for name, stop := range h.runners {
		fmt.Printf("Stopping check %s\n", name)
		close(stop)
	}
	time.Sleep(time.Second)
	// Reset runner map
	h.runners = make(map[string]chan struct{}, 0)

	// Reset states
	h.safeResetStates()

	h.started = false
	return nil
}

func (h *Monitor) State() (map[string]State, error) {
	return h.safeGetStates(), nil
}

func (h *Monitor) startRunner(cfg *Config,
	stop <-chan struct{}) {

	checkFunc := func() {
		data, err := cfg.Checker.Status()

		stateEntry := &State{
			Name:      cfg.Name,
			Status:    "ok",
			Details:   data,
			CheckTime: time.Now(),
		}

		if err != nil {
			fmt.Printf("check %s has failed with error %v\n", cfg.Name, err)
			stateEntry.Err = err.Error()
			stateEntry.Status = "failed"
		}

		h.safeUpdateState(stateEntry)

		if cfg.OnComplete != nil {
			go cfg.OnComplete(stateEntry)
		}
	}

	go func() {

		time.Sleep(time.Duration(h.RandomStartTimeMillis()) * time.Millisecond)
		fmt.Printf("%s Starting check %s\n", time.Now(), cfg.Name)
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		checkFunc()

	RunLoop:
		for {
			select {
			case <-ticker.C:
				checkFunc()
			case <-stop:
				break RunLoop
			}
		}

	}()
}

// resets the states in a concurrency-safe manner
func (h *Monitor) safeResetStates() {
	h.statesLock.Lock()
	defer h.statesLock.Unlock()
	h.states = make(map[string]State, 0)
}

// updates the check state in a concurrency-safe manner
func (h *Monitor) safeUpdateState(stateEntry *State) {
	// dispatch any status listeners
	h.handleStatusListener(stateEntry)

	// update states here
	h.statesLock.Lock()
	defer h.statesLock.Unlock()

	h.states[stateEntry.Name] = *stateEntry
}

// get all states in a concurrency-safe manner
func (h *Monitor) safeGetStates() map[string]State {
	h.statesLock.Lock()
	defer h.statesLock.Unlock()

	// deep copy h.states to avoid race
	statesCopy := make(map[string]State, 0)

	for k, v := range h.states {
		statesCopy[k] = v
	}

	return statesCopy
}

// if a status listener is attached
func (h *Monitor) handleStatusListener(stateEntry *State) {
	// get the previous state
	h.statesLock.Lock()
	prevState := h.states[stateEntry.Name]
	h.statesLock.Unlock()

	// state is failure
	if stateEntry.isFailure() {
		if !prevState.isFailure() {
			// new failure: previous state was ok
			if h.StatusListener != nil {
				go h.StatusListener.CheckFailed(stateEntry)
			}

			stateEntry.TimeOfFirstFailure = time.Now()
		} else {
			// carry the time of first failure from the previous state
			stateEntry.TimeOfFirstFailure = prevState.TimeOfFirstFailure
			if h.StatusListener != nil {
				go h.StatusListener.StillFailing(stateEntry, prevState.ContiguousFailures)
			}
		}
		stateEntry.ContiguousFailures = prevState.ContiguousFailures + 1

	} else if prevState.isFailure() {
		// recovery, previous state was failure
		failureSeconds := time.Now().Sub(prevState.TimeOfFirstFailure).Seconds()

		if h.StatusListener != nil {
			go h.StatusListener.CheckRecovered(stateEntry, prevState.ContiguousFailures, failureSeconds)
		}
	}
}
