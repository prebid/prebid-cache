package backends

import (
	"github.com/gofrs/uuid"
	"testing"
)

var u1 = uuid.Must(uuid.NewV4())

func TestPartitionKey(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	u2, err := uuid.NewV4()
	if err != nil {
		t.Errorf("Error generating version 4 UUID")
	}
	id := u2.String()

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

	u2, err := uuid.NewV4()
	if err != nil {
		t.Errorf("Error generating version 4 UUID")
	}
	id := u2.String()

	expected := "[\"" + id[0:4] + "\"]"

	got := azureTable.wrapForHeader(azureTable.makePartitionKey(id))

	if got != expected {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", expected, got)
	}
}
