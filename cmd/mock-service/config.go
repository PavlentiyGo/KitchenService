package main

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type MockConfig struct {
	RequestTimeout time.Duration `envconfig:"MOCK_REQUEST_TIMEOUT" default:"5s"`
	FailurePercent int           `envconfig:"MOCK_FAILURE_PERCENT" default:"80"`
}

func NewConfig() (*MockConfig, error) {

	config := MockConfig{}
	err := envconfig.Process("", &config)
	if err != nil {
		return &MockConfig{}, err
	}

	return &config, nil
}
func NewConfigMust() *MockConfig {
	config, err := NewConfig()
	if err != nil {
		panic(err)
	}
	return config
}
