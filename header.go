package libpmtiles

import "strconv"

const HEADERV3_LEN_BYTES = 127

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

type PositionE7 struct {
	Lat int32 // Latitude expressed in e7 format
	Lon int32 // Longitude expressed in e7 format
}

// EntryV3 represents a directory entry
type EntryV3 struct {
	TileID    uint64
	Offset    uint64
	Length    uint32
	RunLength uint32
}
