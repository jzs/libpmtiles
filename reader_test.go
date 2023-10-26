package libpmtiles_test

import (
	"libpmtiles"
	"log"
	"testing"
)

func TestReader(t *testing.T) {
	header, err := libpmtiles.Open("/home/jzs/20231025.pmtiles")
	if err != nil {
		t.Fatalf("expect no err, got %v", err)
	}

	log.Printf("%+v", header)
}
