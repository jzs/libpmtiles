package libpmtiles_test

import (
	"libpmtiles"
	"log"
	"testing"
)

func TestReader(t *testing.T) {
	tiles, err := libpmtiles.Open("/home/jzs/20231025.pmtiles")
	if err != nil {
		t.Fatalf("expect no err, got %v", err)
	}

	tiledata, err := tiles.GetTile(1, 1, 1, "mvt")
	if err != nil {
		t.Fatalf("Expect no err, got: %v", err)
	}

	if tiledata == nil {
		t.Fatalf("Expect tile data, got nothing")
	}

	log.Printf("%+v", tiles)
}
