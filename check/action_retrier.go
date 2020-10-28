package check

import (
	"gopkg.in/eapache/go-resiliency.v1/retrier"
	"time"
)

type ActionRetrier struct {
	*retrier.Retrier
}

type ActionRetrierConfig struct {
	NumOfRetries uint8
	Amount       time.Duration
}

func NewActionRetrier(cfg ActionRetrierConfig) ActionRetrier {
	return ActionRetrier{
		retrier.New(
			retrier.ExponentialBackoff(int(cfg.NumOfRetries), cfg.Amount),
			nil,
		),
	}
}
