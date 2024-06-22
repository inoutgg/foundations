package env

import (
	"os"
	"testing"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	Config1 string `env:"CONFIG1"`
	Config2 int    `env:"CONFIG2"`
	Config3 string `env:"CONFIG3"`
}

func TestLoad(t *testing.T) {

	t.Run("missing value", func(t *testing.T) {
		// Make sure that the environment variables are not set.
		os.Clearenv()

		cfg, err := Load[Config]()
		if err != nil {
			t.Fatal(err)
		}

		validate(t, "CONFIG1", "", cfg.Config1)
		validate(t, "CONFIG2", 0, cfg.Config2)
	})

	t.Run("load from env and file", func(t *testing.T) {
		// Make sure that the environment variables are not set.
		os.Clearenv()

		t.Setenv("CONFIG1", "value1")

		cfg, err := Load[Config]("fixtures/test.env")
		if err != nil {
			t.Fatal(err)
		}

		validate(t, "CONFIG1", "value1", cfg.Config1)
		validate(t, "CONFIG2", 2, cfg.Config2)
	})

	t.Run("env variables take precedence over file", func(t *testing.T) {
		// Make sure that the environment variables are not set.
		os.Clearenv()

		t.Setenv("CONFIG2", "7")

		cfg, err := Load[Config]("fixtures/test.env")
		if err != nil {
			t.Fatal(err)
		}

		validate(t, "CONFIG2", 7, cfg.Config2)
		validate(t, "CONFIG3", "value3", cfg.Config3)
	})

	t.Run("it fails on validation", func(t *testing.T) {
		// Make sure that the environment variables are not set.
		os.Clearenv()

		t.Setenv("NON_ZERO_FLOAT", "1")

		type Config struct {
			NonZeroFoat    float32 `env:"NON_ZERO_FLOAT"   validate:"required"`
			NonEmptyString string  `env:"NON_EMPTY_STRING" validate:"required"`
			NonZeroInt     int     `env:"NON_ZERO_INT"     validate:"required"`
		}

		_, err := Load[Config]()
		if err == nil {
			t.Fatal("expected error")
		}

		errs := err.(validator.ValidationErrors)
		if len(errs) != 2 {
			t.Fatalf("expected 2 errors, got %d", len(errs))
		}

		validate(t, "NON_ZERO_INT", "Config.NonEmptyString", errs[0].StructNamespace())
		validate(t, "NON_EMPTY_STRING", "Config.NonZeroInt", errs[1].StructNamespace())
	})
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
