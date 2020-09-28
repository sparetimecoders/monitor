package monitor

import (
	"time"
)

type Config struct {
	// Name of the check
	Name string

	// Checker instance used to perform health check
	Checker Checker

	// Interval between health checks
	Interval time.Duration

	// Hook that gets called when this health check is complete
	OnComplete func(state *State)
}

func NewConfig(name string, checker Checker, interval time.Duration, onComplete func(state *State)) (*Config, error) {
	// TODO Check input
	return &Config{
		Name:       name,
		Checker:    checker,
		Interval:   interval,
		OnComplete: onComplete,
	}, nil
}
