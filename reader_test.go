package libpmtiles_test

import (
	"libpmtiles"
	"testing"
)

func TestReader(t *testing.T) {
	_, err := libpmtiles.Open("/home/jzs/20231025.pmtiles")
	if err != nil {
		t.Fatalf("expect no err, got %v", err)
	}
}
