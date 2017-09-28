package main

import (
	"github.com/satori/go.uuid"
	"testing"
)

func TestPartitionKey(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id := uuid.NewV4().String()
	expected := id[0:4]

	got := azureTable.makePartitionKey(id)

	if got != expected {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", expected, got)
	}
}

func TestShortPartitionKey(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id := "abc"
	got := azureTable.makePartitionKey(id)

	if got != id {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", id, got)
	}
}

func TestEmptyPartitionKey(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id := ""
	got := azureTable.makePartitionKey(id)

	if got != id {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", id, got)
	}
}

func TestPartitionKeyHeader(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id := uuid.NewV4().String()
	expected := "[\"" + id[0:4] + "\"]"

	got := azureTable.wrapForHeader(azureTable.makePartitionKey(id))

	if got != expected {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", expected, got)
	}
}
