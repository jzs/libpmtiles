package libpmtiles_test

import (
	"libpmtiles"
	"log"
	"testing"
)

func TestReader(t *testing.T) {
	tiles, err := libpmtiles.Open("extracts/samsoe_test.pmtiles")
	if err != nil {
		t.Fatalf("expect no err, got %v", err)
	}

	tiledata, err := tiles.GetTile(0, 0, 0, "mvt")
	if err != nil {
		t.Fatalf("Expect no err, got: %v", err)
	}

	if tiledata == nil {
		t.Fatalf("Expect tile data, got nothing")
	}

	if len(tiledata) == 0 {
		t.Fatalf("Expect to find tile data, got 0 bytes back")
	}

	log.Printf("%+v", tiles)
}
