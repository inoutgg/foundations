package env

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Make sure that the environment variables are not set.
	os.Clearenv()

	t.Setenv("CONFIG1", "value1")

	type Config struct {
		Config1 string `env:"CONFIG1"`
		Config2 int    `env:"CONFIG2"`
	}

	cfg, err := Load[Config]("fixtures/test.env")
	if err != nil {
		t.Fatal(err)
	}

	validate(t, "CONFIG1", "value1", cfg.Config1)
	validate(t, "CONFIG2", 2, cfg.Config2)
}

func TestLoadMissingConfigFile(t *testing.T) {
	// Make sure that the environment variables are not set.
	os.Clearenv()

	t.Setenv("CONFIG1", "value1")

	type Config struct {
		Config1 string `env:"CONFIG1"`
		Config2 string `env:"CONFIG2"`
	}

	cfg, err := Load[Config]("fixtures/missing.env")
	if err != nil {
		t.Fatal(err)
	}

	validate(t, "CONFIG1", "value1", cfg.Config1)
	validate(t, "CONFIG2", "", cfg.Config2)
}

func validate[T comparable](t *testing.T, key string, expected T, got T) {
	if expected != got {
		t.Errorf(
			"Mismatch for key '%v', expected '%v', got '%#v'",
			key,
			expected,
			got,
		)
	}
}
