package azure

import (
	"github.com/satori/go.uuid"
	"testing"
)

func TestPartitionKey(t *testing.T) {
	backend, err := NewBackend("abc", "aGprc2NoNzc2MjdlZHVpSHVER1NIQ0pld3lhNzMyNjRlN2ReIyQmI25jc2Fr")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	id := uuid.NewV4().String()
	expected := id[0:4]

	got := backend.makePartitionKey(id)

	if got != expected {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", expected, got)
	}
}

func TestPartitionKeyHeader(t *testing.T) {
	backend, err := NewBackend("abc", "aGprc2NoNzc2MjdlZHVpSHVER1NIQ0pld3lhNzMyNjRlN2ReIyQmI25jc2Fr")
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}

	id := uuid.NewV4().String()
	expected := "[\"" + id[0:4] + "\"]"

	got := backend.wrapForHeader(backend.makePartitionKey(id))

	if got != expected {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", expected, got)
	}
}
