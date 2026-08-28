package testkit

import "testing"

func TestRedisRespondsToPing(t *testing.T) {
	connection := Redis(t)
	if err := connection.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}
