package libpmtiles

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
)

type PMTiles struct {
	header  HeaderV3
	rootDir []EntryV3
	file    *os.File
}

func (pmt *PMTiles) Header() HeaderV3 {
	return pmt.header
}

func Open(path string) (*PMTiles, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Open filepath: %w", err)
	}

	header, rootDir, err := readHeaderAndRootDir(file)
	if err != nil {
		return nil, fmt.Errorf("Read header: %w", err)
	}

	log.Printf("%+v", header)

	return &PMTiles{
		header:  header,
		rootDir: rootDir,
		file:    file,
	}, nil
}

func readHeaderAndRootDir(stream io.ReadSeeker) (HeaderV3, []EntryV3, error) {
	if _, err := stream.Seek(0, 0); err != nil {
		return HeaderV3{}, nil, fmt.Errorf("Seek to start: %w", err)
	}

	headerBytes := make([]byte, HEADERV3_LEN_BYTES)
	if _, err := io.ReadFull(stream, headerBytes); err != nil {
		return HeaderV3{}, nil, fmt.Errorf("Read header bytes: %w", err)
	}

	header, err := deserialize_header(headerBytes)
	if err != nil {
		return HeaderV3{}, nil, fmt.Errorf("Deserializing header: %w", err)
	}

	// do something with header?
	if _, err := stream.Seek(int64(header.RootOffset), 0); err != nil {
		return HeaderV3{}, nil, fmt.Errorf("seeking to root entries offset: %w", err)
	}
	rootEntries, err := deserialize_entries(io.LimitReader(stream, int64(header.RootLength)))
	if err != nil {
		return HeaderV3{}, nil, fmt.Errorf("deserializing root entries: %w", err)
	}

	return header, rootEntries, nil
}

func (pmt *PMTiles) GetTile(z uint8, x, y uint32, extension string) ([]byte, error) {
	if ExtensionToTileType(extension) != pmt.header.TileType {
		return nil, fmt.Errorf("Unsupported Extension")
	}

	if z < pmt.header.MinZoom || z > pmt.header.MaxZoom {
		log.Println("Zoom out of bounds. Tile not found")
		return nil, TileNotFound
	}

	tileID := ZxyToID(z, x, y)

	rootTile, found := findTile(pmt.rootDir, tileID)
	if !found {
		log.Printf("Tile not found with id: %v", tileID)
		return nil, TileNotFound
	}

	if rootTile.RunLength > 0 {
		data, err := pmt.loadTileData(rootTile)
		if err != nil {
			log.Printf("Loading tile data failed for tile: %v", tileID)
			return nil, fmt.Errorf("Loading tile data: %v", err)
		}
		return data, nil
	}

	dirOffset := int64(pmt.header.LeafDirectoriesOffset + rootTile.Offset)
	dirLength := int64(rootTile.Length)

	// TODO: Do not do this infinitely.
	for {
		tile, err := pmt.loadTile(dirOffset, dirLength, tileID)
		if err == TileNotFound {
			return nil, TileNotFound
		}
		if err != nil {
			log.Printf("Failed loading tile for id: %v", tileID)
			return nil, fmt.Errorf("Loading tile: %v", err)
		}

		if tile.RunLength == 0 {
			dirOffset = int64(pmt.header.LeafDirectoriesOffset + tile.Offset)
			dirLength = int64(tile.Length)
			continue
		}

		// Load tiledata...
		tileData, err := pmt.loadTileData(tile)
		if err != nil {
			log.Printf("Failed loading tile data for tile: %v", tileID)
			return nil, fmt.Errorf("Loading tile data: %v", err)
		}
		return tileData, nil
	}

	return nil, TileNotFound
}

var TileNotFound = fmt.Errorf("Tile not found")

func (pmt *PMTiles) loadTile(offset, length int64, tileID uint64) (EntryV3, error) {
	// Seek to directory start
	pmt.file.Seek(offset, 0)

	directory, err := deserialize_entries(io.LimitReader(pmt.file, length))
	if err != nil {
		return EntryV3{}, fmt.Errorf("Deserialize entries: %+w", err)
	}
	tile, found := findTile(directory, tileID)
	if !found {
		return EntryV3{}, TileNotFound
	}
	return tile, nil
}

func (pmt *PMTiles) loadTileData(tile EntryV3) ([]byte, error) {
	pmt.file.Seek(int64(pmt.header.TileDataOffset+tile.Offset), 0)
	lr := io.LimitReader(pmt.file, int64(tile.Length))
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("Read tile data: %w", err)
	}
	return data, nil
}

func findTile(entries []EntryV3, tileId uint64) (EntryV3, bool) {
	m := 0
	n := len(entries) - 1
	for m <= n {
		k := (n + m) >> 1
		cmp := int64(tileId) - int64(entries[k].TileID)
		if cmp > 0 {
			m = k + 1
		} else if cmp < 0 {
			n = k - 1
		} else {
			return entries[k], true
		}
	}

	// at this point, m > n
	if n >= 0 {
		if entries[n].RunLength == 0 {
			return entries[n], true
		}
		if tileId-entries[n].TileID < uint64(entries[n].RunLength) {
			return entries[n], true
		}
	}
	return EntryV3{}, false
}

func deserialize_entries(data io.Reader) ([]EntryV3, error) {
	entries := make([]EntryV3, 0)

	reader, err := gzip.NewReader(data)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	byte_reader := bufio.NewReader(reader)

	num_entries, err := binary.ReadUvarint(byte_reader)
	if err != nil {
		return nil, fmt.Errorf("read num entries: %w", err)
	}

	last_id := uint64(0)
	for i := uint64(0); i < num_entries; i++ {
		tmp, _ := binary.ReadUvarint(byte_reader)
		entries = append(entries, EntryV3{last_id + tmp, 0, 0, 0})
		last_id = last_id + tmp
	}

	for i := uint64(0); i < num_entries; i++ {
		run_length, _ := binary.ReadUvarint(byte_reader)
		entries[i].RunLength = uint32(run_length)
	}

	for i := uint64(0); i < num_entries; i++ {
		length, _ := binary.ReadUvarint(byte_reader)
		entries[i].Length = uint32(length)
	}

	for i := uint64(0); i < num_entries; i++ {
		tmp, _ := binary.ReadUvarint(byte_reader)
		if i > 0 && tmp == 0 {
			entries[i].Offset = entries[i-1].Offset + uint64(entries[i-1].Length)
		} else {
			entries[i].Offset = tmp - 1
		}
	}

	return entries, nil
}

func deserialize_header(d []byte) (HeaderV3, error) {
	h := HeaderV3{}
	magic_number := d[0:7]
	if string(magic_number) != "PMTiles" {
		return h, fmt.Errorf("Magic number not detected. Are you sure this is a PMTiles archive?")
	}

	spec_version := d[7]
	if spec_version > uint8(3) {
		return h, fmt.Errorf("Archive is spec version %d, but this program only supports version 3: upgrade your pmtiles program.", spec_version)
	}

	h.SpecVersion = spec_version
	h.RootOffset = binary.LittleEndian.Uint64(d[8 : 8+8])
	h.RootLength = binary.LittleEndian.Uint64(d[16 : 16+8])
	h.MetadataOffset = binary.LittleEndian.Uint64(d[24 : 24+8])
	h.MetadataLength = binary.LittleEndian.Uint64(d[32 : 32+8])
	h.LeafDirectoriesOffset = binary.LittleEndian.Uint64(d[40 : 40+8])
	h.LeafDirectoriesLength = binary.LittleEndian.Uint64(d[48 : 48+8])
	h.TileDataOffset = binary.LittleEndian.Uint64(d[56 : 56+8])
	h.TileDataLength = binary.LittleEndian.Uint64(d[64 : 64+8])
	h.AddressedTilesCount = binary.LittleEndian.Uint64(d[72 : 72+8])
	h.TileEntriesCount = binary.LittleEndian.Uint64(d[80 : 80+8])
	h.TileContentsCount = binary.LittleEndian.Uint64(d[88 : 88+8])
	h.Clustered = (d[96] == 0x1)
	h.InternalCompression = Compression(d[97])
	h.TileCompression = Compression(d[98])
	h.TileType = TileType(d[99])
	h.MinZoom = d[100]
	h.MaxZoom = d[101]
	h.MinPos = PositionE7{
		Lon: int32(binary.LittleEndian.Uint32(d[102 : 102+4])),
		Lat: int32(binary.LittleEndian.Uint32(d[106 : 106+4])),
	}
	h.MaxPos = PositionE7{
		Lon: int32(binary.LittleEndian.Uint32(d[110 : 110+4])),
		Lat: int32(binary.LittleEndian.Uint32(d[114 : 114+4])),
	}
	h.CenterZoom = d[118]
	h.CenterPos = PositionE7{
		Lon: int32(binary.LittleEndian.Uint32(d[119 : 119+4])),
		Lat: int32(binary.LittleEndian.Uint32(d[123 : 123+4])),
	}

	return h, nil
}
