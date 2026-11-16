package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

const validDotEnv = `HTTP_ADDRESS=:3000
HTTP_READ_TIMEOUT=5s
HTTP_WRITE_TIMEOUT=10s
REDIS_ADDRESS=localhost:6379
REDIS_DB=0
REDIS_TIMEOUT=8s
`

var configKeys = []string{
	"HTTP_ADDRESS", "HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT",
	"REDIS_ADDRESS", "REDIS_USERNAME", "REDIS_PASSWORD", "REDIS_DB", "REDIS_TIMEOUT",
}

// Load reads the working directory and process environment, so these tests must
// not run in parallel. Restore both after each test and never read a real .env.
func prepareConfig(t *testing.T, contents string) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, key := range configKeys {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(".env", []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadParsesDotEnv(t *testing.T) {
	prepareConfig(t, validDotEnv)
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		HTTP:  HTTPConfig{Address: ":3000", ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second},
		Redis: RedisConfig{Address: "localhost:6379", DB: 0, Timeout: 8 * time.Second},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("configuration = %+v, want %+v", got, want)
	}
}

func TestLoadPreservesExistingEnvironment(t *testing.T) {
	prepareConfig(t, validDotEnv)
	t.Setenv("HTTP_ADDRESS", ":4000")
	t.Setenv("HTTP_READ_TIMEOUT", "7s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "9s")
	t.Setenv("REDIS_ADDRESS", "redis:6379")
	t.Setenv("REDIS_DB", "2")
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.HTTP.Address != ":4000" || got.HTTP.ReadTimeout != 7*time.Second || got.HTTP.WriteTimeout != 9*time.Second || got.Redis.Address != "redis:6379" || got.Redis.DB != 2 {
		t.Errorf("existing environment was not preserved: %+v", got)
	}
}

func TestLoadOptionalCredentials(t *testing.T) {
	prepareConfig(t, validDotEnv+"REDIS_USERNAME=example-user\nREDIS_PASSWORD=example-password\n")
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Redis.Username != "example-user" || got.Redis.Password != "example-password" {
		t.Error("Redis credentials were not loaded")
	}
}

func TestLoadRequiresDotEnvFile(t *testing.T) {
	prepareConfig(t, validDotEnv)
	if err := os.Remove(".env"); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "load .env") {
		t.Fatalf("got error %v, want a missing .env error", err)
	}
}

func TestLoadRejectsMalformedDotEnv(t *testing.T) {
	prepareConfig(t, "HTTP_ADDRESS=\"unterminated\n")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "load .env") {
		t.Fatalf("got error %v, want a malformed .env error", err)
	}
}

func TestLoadUsesDefaults(t *testing.T) {
	for _, name := range []string{"missing values", "empty values"} {
		t.Run(name, func(t *testing.T) {
			prepareConfig(t, "")
			if name == "empty values" {
				for _, key := range configKeys {
					t.Setenv(key, "")
				}
			}
			got, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			want := Config{
				HTTP:  HTTPConfig{Address: ":3000", ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second},
				Redis: RedisConfig{Address: "localhost:6379", DB: 0, Timeout: 5 * time.Second},
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("configuration = %+v, want defaults %+v", got, want)
			}
		})
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		key, value, errorField string
	}{
		{"HTTP_READ_TIMEOUT", "five seconds", "ReadTimeout"},
		{"HTTP_WRITE_TIMEOUT", "10", "WriteTimeout"},
		{"HTTP_READ_TIMEOUT", "0s", "HTTP_READ_TIMEOUT"},
		{"HTTP_WRITE_TIMEOUT", "-1s", "HTTP_WRITE_TIMEOUT"},
		{"REDIS_DB", "not-an-integer", "DB"},
		{"REDIS_DB", "1.5", "DB"},
		{"REDIS_DB", "-1", "REDIS_DB"},
		{"REDIS_TIMEOUT", "invalid", "Timeout"},
		{"REDIS_TIMEOUT", "0s", "REDIS_TIMEOUT"},
		{"REDIS_TIMEOUT", "-1s", "REDIS_TIMEOUT"},
	} {
		t.Run(tc.key+"/"+tc.value, func(t *testing.T) {
			prepareConfig(t, validDotEnv)
			t.Setenv(tc.key, tc.value)
			cfg, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.errorField) {
				t.Fatalf("got error %v, want an error naming %s", err, tc.errorField)
			}
			if !reflect.DeepEqual(cfg, Config{}) {
				t.Error("Load returned a partially populated config on failure")
			}
		})
	}
}
