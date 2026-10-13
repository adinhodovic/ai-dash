package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewer(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"0.7.0", "0.8.0", true},
		{"0.8.0", "0.8.0", false},
		{"0.9.0", "0.8.0", false},
		{"0.8.9", "0.10.0", true},
		{"v0.8.0", "0.8.1", true},
		{"dev", "0.8.0", true},
		{"0.8.0-3-gabc", "0.8.0", true},
		{"0.8.0", "nightly", false},
	}
	for _, tt := range tests {
		if got := Newer(tt.current, tt.latest); got != tt.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestIsReleaseVersion(t *testing.T) {
	for v, want := range map[string]bool{
		"0.8.0":                              true,
		"v0.8.0":                             true,
		"dev":                                false,
		"0.8.0-3-gea3bbeb-dirty":             false,
		"v0.0.0-20260920121937-0165a0ee6fc7": false,
	} {
		if got := IsReleaseVersion(v); got != want {
			t.Errorf("IsReleaseVersion(%q) = %v, want %v", v, got, want)
		}
	}
}

func newServer(t *testing.T, binary []byte, sums string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		rel := Release{Tag: "0.9.0", Assets: []Asset{
			{Name: "ai-dash-linux-amd64", URL: srv.URL + "/bin"},
		}}
		if sums != "" {
			rel.Assets = append(rel.Assets, Asset{Name: ChecksumsAsset, URL: srv.URL + "/sums"})
		}
		_ = json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/bin", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(binary) })
	mux.HandleFunc(
		"/sums",
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(sums)) },
	)
	return srv
}

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func download(t *testing.T, srv *httptest.Server) ([]byte, error) {
	t.Helper()
	c := Client{HTTP: srv.Client(), APIBase: srv.URL}
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Tag != "0.9.0" {
		t.Fatalf("tag = %q", rel.Tag)
	}
	return c.Download(context.Background(), rel, "ai-dash-linux-amd64")
}

func TestDownloadVerifiesChecksum(t *testing.T) {
	bin := []byte("new binary")
	srv := newServer(t, bin, sha(bin)+"  ai-dash-linux-amd64\nabc  ai-dash-darwin-arm64\n")
	data, err := download(t, srv)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(bin) {
		t.Fatalf("data = %q", data)
	}
}

func TestDownloadChecksumMismatch(t *testing.T) {
	srv := newServer(t, []byte("tampered"), sha([]byte("original"))+"  ai-dash-linux-amd64\n")
	if _, err := download(t, srv); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestDownloadWithoutChecksums(t *testing.T) {
	srv := newServer(t, []byte("bin"), "")
	data, err := download(t, srv)
	if !errors.Is(err, ErrNoChecksum) {
		t.Fatalf("err = %v, want ErrNoChecksum", err)
	}
	if string(data) != "bin" {
		t.Fatalf("data = %q", data)
	}
}

func TestReplace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-dash")
	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Replace(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("content = %q", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v", info.Mode())
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}
