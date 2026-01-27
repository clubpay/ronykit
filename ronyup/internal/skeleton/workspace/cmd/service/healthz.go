package main

import "github.com/clubpay/ronykit/rony"

type HealthInput struct{}

type HealthOutput struct {
	Ok bool `json:"ok"`
}

func Healthz(_ *rony.UnaryCtx[rony.EMPTY, rony.NOP], _ HealthInput) (*HealthOutput, error) { //nolint:unparam
	return &HealthOutput{Ok: true}, nil
}
