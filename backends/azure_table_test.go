package backends

import (
	"github.com/prebid/prebid-cache/utils"
	"testing"
)

func TestPartitionKey(t *testing.T) {
	azureTable := NewAzureBackend("abc", "def")

	id, err := utils.GenerateRandomId()
	if err != nil {
		t.Errorf("Error generating version 4 UUID")
	}

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

	id, err := utils.GenerateRandomId()
	if err != nil {
		t.Errorf("Error generating version 4 UUID")
	}

	expected := "[\"" + id[0:4] + "\"]"

	got := azureTable.wrapForHeader(azureTable.makePartitionKey(id))

	if got != expected {
		t.Errorf("Bad partition key. Expected: %s, Got: %s", expected, got)
	}
}
