package libpmtiles

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

const HEADERV3_LEN_BYTES = 127

type PMTiles struct {
	header  HeaderV3
	rootDir []EntryV3
	file    *os.File
}

func Open(path string) (*PMTiles, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Open filepath: %w", err)
	}

	header, rootDir, err := readHeader(file)
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

func readHeader(stream io.ReadSeeker) (HeaderV3, []EntryV3, error) {
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

func (pmt *PMTiles) GetTile(z uint8, x, y uint32) {
	tileID := ZxyToId(z, x, y)

	// Seek to directory start
	pmt.file.Seek(int64(pmt.header.RootOffset), 0)

	directory, err := deserialize_entries(io.LimitReader(pmt.file, int64(pmt.header.RootLength)))
	if err != nil {
		panic("BOOM")
	}
	tile, found := findTile(directory, tileID)
	if !found {
		panic("BOOM tile not found")
	}
	log.Println(tile)

	if tile.RunLength > 0 {
		// range reader etc...
		// etc...
	} else {
		// Try look up leaf?
	}

	// Found tile. Now load data.
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

func ZxyToId(z uint8, x uint32, y uint32) uint64 {
	var acc uint64
	var tz uint8
	for ; tz < z; tz++ {
		acc += (0x1 << tz) * (0x1 << tz)
	}
	var n uint64 = 1 << z
	var rx uint64
	var ry uint64
	var d uint64
	tx := uint64(x)
	ty := uint64(y)
	for s := n / 2; s > 0; s /= 2 {
		if tx&s > 0 {
			rx = 1
		} else {
			rx = 0
		}
		if ty&s > 0 {
			ry = 1
		} else {
			ry = 0
		}
		d += s * s * ((3 * rx) ^ ry)
		rotate(s, &tx, &ty, rx, ry)
	}
	return acc + d
}

func rotate(n uint64, x *uint64, y *uint64, rx uint64, ry uint64) {
	if ry == 0 {
		if rx == 1 {
			*x = n - 1 - *x
			*y = n - 1 - *y
		}
		*x, *y = *y, *x
	}
}

type Compression uint8

func (c Compression) String() string {
	switch c {
	case UnknownCompression:
		return "Unknown compression"
	case NoCompression:
		return "No compression"
	case Gzip:
		return "gzip"
	case Brotli:
		return "brotli"
	case Zstd:
		return "zstd"
	default:
		return strconv.Itoa(int(c))
	}
}

const (
	UnknownCompression Compression = 0
	NoCompression                  = 1
	Gzip                           = 2
	Brotli                         = 3
	Zstd                           = 4
)

type TileType uint8

func (t TileType) String() string {
	switch t {
	case UnknownTileType:
		return "unkown tile type"
	case Mvt:
		return "mvt"
	case Png:
		return "png"
	case Jpeg:
		return "jpeg"
	case Webp:
		return "webp"
	case Avif:
		return "avif"
	default:
		return strconv.Itoa(int(t))
	}
}

const (
	UnknownTileType TileType = 0
	Mvt                      = 1
	Png                      = 2
	Jpeg                     = 3
	Webp                     = 4
	Avif                     = 5
)

// HeaderV3
type HeaderV3 struct {
	SpecVersion           uint8       // SpecVersion should be 0x03 since we only implement v3
	RootOffset            uint64      // RootOffset is the offset of the root directory. It's relative to the first byte of the archive
	RootLength            uint64      // RootLength specifies the number of bytes in the root directory
	MetadataOffset        uint64      //MetadataOffset is the offset of the metadata
	MetadataLength        uint64      //MetadataLength is the number of bytes of metadata
	LeafDirectoriesOffset uint64      //LeafDirectoriesOffset offset of leaf directories
	LeafDirectoriesLength uint64      // LeafDirectoriesLength length of leaf directories in bytes. 0 Means no leaf directories in archive.
	TileDataOffset        uint64      // TileDataOffset offset of the first byte of tiledata
	TileDataLength        uint64      // TileDataLength is length of bytes of tiledata
	AddressedTilesCount   uint64      //AddressedTilesCount The Number of Addressed Tiles is an 8-byte field specifying the total number of tiles in the PMTiles archive, before RunLength Encoding. A value of 0 indicates that the number is unknown.
	TileEntriesCount      uint64      // TileEntriesCount The Number of Tile Entries is an 8-byte field specifying the total number of tile entries: entries where RunLength is greater than 0. A value of 0 indicates that the number is unknown.
	TileContentsCount     uint64      // TileContentsCount The Number of Tile Contents is an 8-byte field specifying the total number of blobs in the tile data section. A value of 0 indicates that the number is unknown.
	Clustered             bool        // Clustered is a 1-byte field specifying if the data of the individual tiles in the data section is ordered by their TileID (clustered) or not (not clustered). Therefore, Clustered means that: *offsets are either contiguous with the previous offset+length, or refer to a lesser offset when writing with deduplication. * the first tile entry in the directory has offset 0. 0x00 == not clustered, 0x01 == clustered
	InternalCompression   Compression // The Internal Compression is a 1-byte field specifying the compression of the root directory, metadata, and all leaf directories.
	TileCompression       Compression // The Tile Compression is a 1-byte field specifying the compression of all tiles.
	TileType              TileType    // The Tile Type is a 1-byte field specifying the type of tiles. 0x00 = unknown, 0x01 = mvt vector tile, 0x02 = png, 0x03 = jpeg, 0x04 = webp, 0x05 = avif
	MinZoom               uint8       // The Min Zoom is a 1-byte field specifying the minimum zoom of the tiles.
	MaxZoom               uint8       // The Max Zoom is a 1-byte field specifying the maximum zoom of the tiles. It must be greater than or equal to the min zoom.
	MinPos                PositionE7  //The Min Position is an 8-byte field that includes the minimum latitude and minimum longitude of the bounds.
	MaxPos                PositionE7  // The Max Position is an 8-byte field including the maximum latitude and maximum longitude of the bounds.
	CenterZoom            uint8       // The Center Zoom is a 1-byte field specifying the center zoom (LOD) of the tiles. A reader MAY use this as the initial zoom when displaying tiles from the PMTiles archive.
	CenterPos             PositionE7
}

type PositionE7 struct {
	Lat int32 // Latitude expressed in e7 format
	Lon int32 // Longitude expressed in e7 format
}

type EntryV3 struct {
	TileID    uint64
	Offset    uint64
	Length    uint32
	RunLength uint32
}
