package testkit

import (
	"bufio"
	"errors"
	"net"
	"os"
	"testing"
	"time"
)

type RedisConnection struct{ connection net.Conn }

func Redis(t *testing.T) *RedisConnection {
	t.Helper()
	address := os.Getenv("TCOMPUTE_TEST_REDIS_ADDRESS")
	if address == "" {
		address = "127.0.0.1:56379"
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			result := &RedisConnection{connection: connection}
			if err := result.Ping(); err == nil {
				t.Cleanup(func() { _ = connection.Close() })
				return result
			}
			_ = connection.Close()
		}
		if time.Now().After(deadline) {
			t.Fatalf("test Redis did not become ready at %s", address)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func (connection *RedisConnection) Ping() error {
	if _, err := connection.connection.Write([]byte("*1\r\n$4\r\nPING\r\n")); err != nil {
		return err
	}
	_ = connection.connection.SetReadDeadline(time.Now().Add(2 * time.Second))
	line, err := bufio.NewReader(connection.connection).ReadString('\n')
	if err != nil {
		return err
	}
	if line != "+PONG\r\n" {
		return errors.New("unexpected Redis PING response")
	}
	return nil
}
