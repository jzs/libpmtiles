package libpmtiles

// ZxyToID converts standard (z,x,y) coordinates to a pmtiles tileID
func ZxyToID(z uint8, x uint32, y uint32) uint64 {
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
