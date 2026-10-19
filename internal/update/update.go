// Package update replaces the running ai-dash binary with the latest GitHub
// release.
package update

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	DefaultAPIBase = "https://api.github.com"
	Repo           = "adinhodovic/ai-dash"
	ChecksumsAsset = "checksums.txt"
)

// ErrNoChecksum is returned by Download when the release publishes no
// checksum for the asset. The data is still returned.
var ErrNoChecksum = errors.New("release has no checksum for asset")

var releaseVersion = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

type Release struct {
	Tag    string  `json:"tag_name"`
	Assets []Asset `json:"assets"`
}

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func (r Release) Asset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

type Client struct {
	HTTP    *http.Client
	APIBase string
}

func NewClient() Client {
	return Client{HTTP: http.DefaultClient, APIBase: DefaultAPIBase}
}

// AssetName is the release binary name for a platform, matching
// `make build-all`.
func AssetName(goos, goarch string) string {
	return fmt.Sprintf("ai-dash-%s-%s", goos, goarch)
}

// IsReleaseVersion reports whether v looks like a tagged release build rather
// than a source build (git describe suffixes, pseudo-versions, "dev").
func IsReleaseVersion(v string) bool {
	return releaseVersion.MatchString(v)
}

// Newer reports whether latest is a higher version than current. Unparsable
// current versions are treated as older.
func Newer(current, latest string) bool {
	cur, okCur := parseVersion(current)
	lat, okLat := parseVersion(latest)
	if !okLat {
		return false
	}
	if !okCur {
		return true
	}
	for i := range lat {
		if lat[i] != cur[i] {
			return lat[i] > cur[i]
		}
	}
	return false
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	if !IsReleaseVersion(v) {
		return out, false
	}
	for i, part := range strings.Split(strings.TrimPrefix(v, "v"), ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func (c Client) Latest(ctx context.Context) (Release, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", c.APIBase, Repo)
	body, err := c.get(ctx, url)
	if err != nil {
		return Release{}, err
	}
	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		return Release{}, fmt.Errorf("decode release: %w", err)
	}
	return rel, nil
}

// Download fetches the named asset and verifies it against the release's
// checksums.txt. It returns the data with ErrNoChecksum when the release has
// no checksum for the asset.
func (c Client) Download(ctx context.Context, rel Release, name string) ([]byte, error) {
	asset, ok := rel.Asset(name)
	if !ok {
		return nil, fmt.Errorf("release %s has no binary %s", rel.Tag, name)
	}
	data, err := c.get(ctx, asset.URL)
	if err != nil {
		return nil, err
	}
	sumsAsset, ok := rel.Asset(ChecksumsAsset)
	if !ok {
		return data, ErrNoChecksum
	}
	sums, err := c.get(ctx, sumsAsset.URL)
	if err != nil {
		return nil, err
	}
	want, ok := checksum(sums, name)
	if !ok {
		return data, ErrNoChecksum
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != want {
		return nil, fmt.Errorf("checksum mismatch for %s", name)
	}
	return data, nil
}

// checksum finds name in sha256sum-formatted output.
func checksum(sums []byte, name string) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(sums))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return strings.ToLower(fields[0]), true
		}
	}
	return "", false
}

func (c Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Replace atomically swaps the binary at path for data. The temp file lives in
// the same directory so the rename stays on one filesystem.
func Replace(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ai-dash-update-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
