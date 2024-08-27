// Package env provides a simple way to load environment variables into a struct.
package env

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	dotenv "github.com/joho/godotenv"
	"go.inout.gg/foundations/must"
)

var (
	// Validator is the default validator used to validate the configuration.
	Validator = validator.New(validator.WithRequiredStructEnabled())
)

// Load loads the environment configuration into a struct T.
//
// By default if no paths are provided, it will look for a file called .env.
// If no file is found, it will simply ignore it.
//
// Make sure to use the `env` tag from the github.com/caarlos0/env/v11 package,
// to specify the environment variable name.
//
// The populated struct is validated using `Validator`.
func Load[T any](paths ...string) (*T, error) {
	var config T

	// Try to load an .env file.
	_ = dotenv.Load(paths...)

	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("env: failed to load environment configuration: %w", err)
	}

	if err := Validator.Struct(config); err != nil {
		return nil, err
	}

	return &config, nil
}

// MustLoad is like Load, but panics if an error occurs.
func MustLoad[T any](paths ...string) *T {
	return must.Must(Load[T](paths...))
}
