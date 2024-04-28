package config

import (
	"errors"
	"strings"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

// Global variable but private
var config *koanf.Koanf = nil

// Load config froom environment. Something different than that could be create
// an overthinking of the structure for a container because we should also
// consider volumes to insert config file.
// Every env var is coverted to lowercase and plitted by underscore "_".
//
// Example: `DATABASE_DSN` becomes `database.dsn`
func LoadConfig() error {
	k := koanf.New(".")

	if err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	}), nil); err != nil {
		return err
	}

	config = k
	return nil
}

// Return the instance or error if the config is not laoded yet
func GetConfig() (*koanf.Koanf, error) {
	if config == nil {
		return nil, errors.New("You must call `InitDb()` first.")
	}
	return config, nil
}
