package terrain50

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	cacheMu sync.RWMutex
	cache   = map[string][]float32{}
)

// five500 maps 500km grid square to (e-index, n-index)
var five500 = map[byte][2]int{'S': {0, 0}, 'T': {1, 0}, 'N': {0, 1}, 'O': {1, 1}, 'H': {0, 2}}

// five500Rev maps (e-index, n-index) to 500km letter
var five500Rev = map[[2]int]byte{{0, 0}: 'S', {1, 0}: 'T', {0, 1}: 'N', {1, 1}: 'O', {0, 2}: 'H'}

const gridLetters = "ABCDEFGHJKLMNOPQRSTUVWXYZ"

// ElevationAt returns OS Terrain 50 elevation in metres at WGS84 lat/lon.
// dataDir is the path to the terr50_gagg_gb directory.
func ElevationAt(dataDir string, lat, lon float64) (float64, error) {
	e, n := wgs84ToBNG(lat, lon)
	return lookupBNG(dataDir, e, n)
}

func lookupBNG(dataDir string, e, n float64) (float64, error) {
	if e < 0 || e >= 700000 || n < 0 || n >= 1300000 {
		return 0, fmt.Errorf("outside GB grid")
	}
	code := bngToTileCode(e, n)
	grid, err := loadTile(dataDir, code)
	if err != nil {
		return 0, err
	}
	const (
		cellSize = 50.0
		nCols    = 200
		nRows    = 200
	)
	tileE, tileN := tileOrigin(code)
	col := int((e - tileE) / cellSize)
	row := int((n - tileN) / cellSize)
	if col < 0 || col >= nCols || row < 0 || row >= nRows {
		return 0, fmt.Errorf("point outside tile %s", code)
	}
	// ASC rows are stored north-first (top = highest northing)
	rowIdx := nRows - 1 - row
	val := float64(grid[rowIdx*nCols+col])
	if val == -9999 {
		return 0, fmt.Errorf("no data for %s", code)
	}
	return val, nil
}

func loadTile(dataDir, code string) ([]float32, error) {
	upper := strings.ToUpper(code)
	lower := strings.ToLower(code)

	cacheMu.RLock()
	if g, ok := cache[upper]; ok {
		cacheMu.RUnlock()
		return g, nil
	}
	cacheMu.RUnlock()

	pattern := filepath.Join(dataDir, "data", lower[:2], lower+"_OST50GRID_*.zip")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("no tile for %s", code)
	}

	zr, err := zip.OpenReader(matches[0])
	if err != nil {
		return nil, fmt.Errorf("opening tile %s: %w", matches[0], err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if strings.ToLower(filepath.Ext(f.Name)) == ".asc" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			grid, err := parseASC(rc)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("parsing tile %s: %w", code, err)
			}
			cacheMu.Lock()
			cache[upper] = grid
			cacheMu.Unlock()
			return grid, nil
		}
	}
	return nil, fmt.Errorf("no .asc in tile %s", code)
}

func parseASC(r io.Reader) ([]float32, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 512*1024), 512*1024)

	var nCols, nRows int
	grid := make([]float32, 0, 200*200)
	inHeader := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if inHeader {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				switch strings.ToLower(fields[0]) {
				case "ncols":
					nCols, _ = strconv.Atoi(fields[1])
					continue
				case "nrows":
					nRows, _ = strconv.Atoi(fields[1])
					continue
				case "xllcorner", "xllcenter", "yllcorner", "yllcenter", "cellsize", "nodata_value":
					continue
				}
			}
			inHeader = false
		}
		for _, tok := range strings.Fields(line) {
			v, err := strconv.ParseFloat(tok, 32)
			if err != nil {
				return nil, fmt.Errorf("bad value %q", tok)
			}
			grid = append(grid, float32(v))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	expected := nCols * nRows
	if len(grid) != expected {
		return nil, fmt.Errorf("expected %d values got %d", expected, len(grid))
	}
	return grid, nil
}

// bngToTileCode converts BNG easting/northing to a 4-character OS tile code (e.g. "SW21").
func bngToTileCode(e, n float64) string {
	ei, ni := int(e), int(n)

	l1 := five500Rev[[2]int{ei / 500000, ni / 500000}]

	e100 := (ei % 500000) / 100000
	n100 := (ni % 500000) / 100000
	l2 := gridLetters[(4-n100)*5+e100]

	ed := (ei % 100000) / 10000
	nd := (ni % 100000) / 10000

	return fmt.Sprintf("%c%c%d%d", l1, l2, ed, nd)
}

// tileOrigin returns the SW corner easting/northing of the named tile.
func tileOrigin(code string) (float64, float64) {
	code = strings.ToUpper(code)
	en500 := five500[code[0]]
	idx := strings.IndexByte(gridLetters, code[1])
	col, row := idx%5, idx/5
	ed := int(code[2] - '0')
	nd := int(code[3] - '0')
	e := float64(en500[0]*500000 + col*100000 + ed*10000)
	n := float64(en500[1]*500000 + (4-row)*100000 + nd*10000)
	return e, n
}

// wgs84ToBNG converts WGS84 lat/lon (degrees) to OSGB36 BNG easting/northing (metres).
// Uses the OS Helmert approximation (±3-5 m accuracy, adequate for 50 m grid cells).
func wgs84ToBNG(latDeg, lonDeg float64) (float64, float64) {
	lat := latDeg * math.Pi / 180
	lon := lonDeg * math.Pi / 180

	// WGS84 ellipsoid
	const aW = 6378137.000
	const fW = 1.0 / 298.257223563
	bW := aW * (1 - fW)
	e2W := 1 - (bW/aW)*(bW/aW)

	sinLat, cosLat := math.Sin(lat), math.Cos(lat)
	sinLon, cosLon := math.Sin(lon), math.Cos(lon)
	nuW := aW / math.Sqrt(1-e2W*sinLat*sinLat)

	// WGS84 → 3D Cartesian
	x := nuW * cosLat * cosLon
	y := nuW * cosLat * sinLon
	z := nuW * (1 - e2W) * sinLat

	// Helmert: WGS84 → OSGB36
	const (
		tx          = -446.448
		ty          = +125.157
		tz          = -542.060
		s           = 20.4894e-6
		arcsecToRad = math.Pi / 648000.0
	)
	rx := -0.1502 * arcsecToRad
	ry := -0.2470 * arcsecToRad
	rz := -0.8421 * arcsecToRad

	x2 := tx + (1+s)*x - rz*y + ry*z
	y2 := ty + rz*x + (1+s)*y - rx*z
	z2 := tz - ry*x + rx*y + (1+s)*z

	// 3D Cartesian → OSGB36 lat/lon (Airy 1830)
	const (
		aA = 6377563.396
		bA = 6356256.909
	)
	e2A := 1 - (bA/aA)*(bA/aA)

	p := math.Sqrt(x2*x2 + y2*y2)
	lat2 := math.Atan2(z2, p*(1-e2A))
	for i := 0; i < 10; i++ {
		nuA := aA / math.Sqrt(1-e2A*math.Sin(lat2)*math.Sin(lat2))
		lat2 = math.Atan2(z2+e2A*nuA*math.Sin(lat2), p)
	}
	lon2 := math.Atan2(y2, x2)

	// OSGB36 lat/lon → BNG Transverse Mercator
	const (
		n0   = -100000.0
		e0   = 400000.0
		f0   = 0.9996012717
		phi0 = 49.0 * math.Pi / 180.0
		lam0 = -2.0 * math.Pi / 180.0
	)
	nA := (aA - bA) / (aA + bA)

	sinPhi := math.Sin(lat2)
	cosPhi := math.Cos(lat2)
	tanPhi := math.Tan(lat2)
	tan2 := tanPhi * tanPhi
	tan4 := tan2 * tan2

	nu := aA * f0 / math.Sqrt(1-e2A*sinPhi*sinPhi)
	rho := aA * f0 * (1 - e2A) / math.Pow(1-e2A*sinPhi*sinPhi, 1.5)
	eta2 := nu/rho - 1

	m := meridionalArc(bA, f0, phi0, lat2, nA)

	I := m + n0
	II := nu / 2 * sinPhi * cosPhi
	III := nu / 24 * sinPhi * math.Pow(cosPhi, 3) * (5 - tan2 + 9*eta2)
	IIIA := nu / 720 * sinPhi * math.Pow(cosPhi, 5) * (61 - 58*tan2 + tan4)
	IV := nu * cosPhi
	V := nu / 6 * math.Pow(cosPhi, 3) * (nu/rho - tan2)
	VI := nu / 120 * math.Pow(cosPhi, 5) * (5 - 18*tan2 + tan4 + 14*eta2 - 58*tan2*eta2)

	dL := lon2 - lam0

	northing := I + II*dL*dL + III*math.Pow(dL, 4) + IIIA*math.Pow(dL, 6)
	easting := e0 + IV*dL + V*math.Pow(dL, 3) + VI*math.Pow(dL, 5)
	return easting, northing
}

// HaversineM returns the great-circle distance in metres between two WGS84 points.
func HaversineM(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000.0
	r := math.Pi / 180
	dLat := (lat2 - lat1) * r
	dLon := (lon2 - lon1) * r
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*r)*math.Cos(lat2*r)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Asin(math.Sqrt(a))
}

// LoSResult holds the outcome of a terrain line-of-sight check.
type LoSResult struct {
	Clear  bool
	ObsLat float64 // WGS84 latitude of first obstruction; zero when Clear
	ObsLon float64 // WGS84 longitude of first obstruction; zero when Clear
}

// LoS reports whether two antenna tips have radio line-of-sight.
// lat/lon in WGS84 degrees; h1/h2 = total height above sea level in metres (QNH + QNF).
// Samples terrain every 100 m. Points outside the GB grid are skipped (benefit of doubt).
// Uses 4/3 effective Earth radius to model atmospheric refraction.
func LoS(dataDir string, lat1, lon1, h1, lat2, lon2, h2 float64) bool {
	return LoSCheck(dataDir, lat1, lon1, h1, lat2, lon2, h2).Clear
}

// LoSCheck is like LoS but also returns the WGS84 coordinates of the first terrain obstruction.
func LoSCheck(dataDir string, lat1, lon1, h1, lat2, lon2, h2 float64) LoSResult {
	const (
		sampleM = 100.0
		rEff    = 4.0 / 3.0 * 6371000.0
	)
	D := HaversineM(lat1, lon1, lat2, lon2)
	if D < 1 {
		return LoSResult{Clear: true}
	}
	n := int(D/sampleM) + 1
	if n < 2 {
		n = 2
	}
	for i := 1; i < n; i++ {
		t := float64(i) / float64(n)
		lat := lat1 + t*(lat2-lat1)
		lon := lon1 + t*(lon2-lon1)
		lineH := h1 + t*(h2-h1)
		bulge := D * D * t * (1 - t) / (2 * rEff)
		elev, err := ElevationAt(dataDir, lat, lon)
		if err != nil {
			continue // outside GB or no data — assume clear
		}
		if elev > lineH-bulge {
			return LoSResult{Clear: false, ObsLat: lat, ObsLon: lon}
		}
	}
	return LoSResult{Clear: true}
}

// meridionalArc computes the meridional arc from phi0 to phi using the OS series formula.
func meridionalArc(b, f0, phi0, phi, n float64) float64 {
	n2 := n * n
	n3 := n2 * n
	dp := phi - phi0
	sp := phi + phi0
	return b * f0 * ((1+n+5.0/4*n2+5.0/4*n3)*dp -
		(3*n+3*n2+21.0/8*n3)*math.Sin(dp)*math.Cos(sp) +
		(15.0/8*n2+15.0/8*n3)*math.Sin(2*dp)*math.Cos(2*sp) -
		(35.0/24*n3)*math.Sin(3*dp)*math.Cos(3*sp))
}
