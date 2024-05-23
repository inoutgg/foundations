// Package env provides a simple way to load environment variables into a struct.
package env

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	dotenv "github.com/joho/godotenv"
	"go.inout.gg/common/must"
)

// Load loads the environment configuration into a struct T.
//
// By default if no paths are provided, it will look for a file called .env.
//
// Make sure to use the `envPrefix` tag from the github.com/caarlos0/env/v10 package,
// to specify the environment variable name.
func Load[T any](paths ...string) (*T, error) {
	var config T

	// Try to load an .env file.
	if err := dotenv.Load(paths...); err != nil {
		return nil, fmt.Errorf("env: failed to load env file: %w", err)
	}

	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("env: failed to load environment configuration: %w", err)
	}

	return &config, nil
}

// MustLoad is like Load, but panics if an error occurs.
func MustLoad[T any](paths ...string) *T {
	return must.Must(Load[T](paths...))
}
