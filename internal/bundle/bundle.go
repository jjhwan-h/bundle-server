package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
)

// regular bundle(전체번들)
type Bundle struct {
	DirPath string
	Oldest  *Version
	Latest  *Version

	etag string // 가장 최신 번들 해시값

	mu sync.RWMutex
}

type Version struct {
	Major int
	Minor int8 // 0~9

	mu sync.Mutex
}

type ReadOnlyVersion interface {
	GetMajor() int
	GetMinor() int8
}

var versionPattern = regexp.MustCompile(`^regular-v(\d+)\.(\d+).tar.gz$`)

func NewBundle(dirPath string) *Bundle {
	versions, err := extractVersionsFromDir(dirPath)
	if err != nil || len(versions) == 0 {
		return &Bundle{
			Latest: &Version{
				Major: 0,
				Minor: 0,
			},
			Oldest: &Version{
				Major: 0,
				Minor: 0,
			},
			DirPath: dirPath,
		}
	}

	return &Bundle{
		Latest:  latestVersion(versions).(*Version),
		Oldest:  oldestVersion(versions).(*Version),
		DirPath: dirPath,
	}
}

func extractVersionsFromDir(dirPath string) ([]ReadOnlyVersion, error) {
	var versions []ReadOnlyVersion

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.Type().IsRegular() {
			return nil // skip non-files
		}

		fileName := d.Name()
		matches := versionPattern.FindStringSubmatch(fileName)
		if len(matches) == 3 {
			major, _ := strconv.Atoi(matches[1])
			minor, _ := strconv.Atoi(matches[2])
			versions = append(versions, &Version{Major: major, Minor: int8(minor)})
		}

		return nil
	})

	return versions, err
}

func latestVersion(versions []ReadOnlyVersion) ReadOnlyVersion {
	if len(versions) == 0 {
		return nil
	}

	max := versions[0]
	for _, v := range versions[1:] {
		if max.GetMajor() < v.GetMajor() ||
			(max.GetMajor() == v.GetMajor() && max.GetMinor() < v.GetMinor()) {
			max = v
		}
	}
	return max
}

func oldestVersion(versions []ReadOnlyVersion) ReadOnlyVersion {
	if len(versions) == 0 {
		return nil
	}

	min := versions[0]
	for _, v := range versions[1:] {
		if min.GetMajor() > v.GetMajor() ||
			(min.GetMajor() == v.GetMajor() && min.GetMinor() > v.GetMinor()) {
			min = v
		}
	}
	return min
}

func (b *Bundle) ETagFromFile() (string, error) {
	p := fmt.Sprintf("%s/regular-v%d.%d.tar.gz", b.DirPath, b.Latest.Major, b.Latest.Minor)
	f, err := os.Open(p)
	if err != nil {
		return "", fmt.Errorf("failed to open bundle: regular-v%d.%d.tar.gz : %w", b.Latest.Major, b.Latest.Minor, err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to hash bundle: regular-v%d.%d.tar.gz : %w", b.Latest.Major, b.Latest.Minor, err)
	}

	sum := hasher.Sum(nil)
	b.etag = `"` + hex.EncodeToString(sum) + `"`
	return `"` + hex.EncodeToString(sum) + `"`, nil // ETag는 따옴표 포함
}

func (b *Bundle) GetEtag() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.etag
}

func (v *Version) IncrementVersion() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.Minor < 9 {
		v.Minor++
	} else {
		v.Major++
		v.Minor = 0
	}
}

func (v *Version) NextVersion() (int, int8) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.Minor < 9 {
		return v.Major, v.Minor + 1
	}
	return v.Major + 1, 0
}

func (v *Version) GetMajor() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.Major
}

func (v *Version) GetMinor() int8 {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.Minor
}
