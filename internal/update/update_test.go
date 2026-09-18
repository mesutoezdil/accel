package update

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// release builds a Fetch over a fake set of release files.
func release(files map[string]string) Fetch {
	return func(url string) ([]byte, error) {
		b, ok := files[url]
		if !ok {
			return nil, fmt.Errorf("%s: 404 Not Found", url)
		}
		return []byte(b), nil
	}
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// fleet is a fake release of one version that publishes the asset for
// whatever machine the test runs on.
func fleet(t *testing.T, tag, body string) (map[string]string, string) {
	t.Helper()
	asset, err := Asset()
	if err != nil {
		t.Fatal(err)
	}
	base := downloadURL + "/" + tag + "/"
	return map[string]string{
		latestURL:              `{"tag_name": "` + tag + `", "prerelease": false}`,
		releasesURL:            `[{"tag_name": "` + tag + `", "prerelease": true}]`,
		base + asset:           body,
		base + "checksums.txt": sum(body) + "  " + asset + "\n" + sum("other") + "  siltide-plan9-386\n",
	}, asset
}

func TestRunInstallsAVerifiedBinary(t *testing.T) {
	files, _ := fleet(t, "v0.4.0", "#!/bin/sh\nnewer\n")
	path := filepath.Join(t.TempDir(), "siltide")
	if err := os.WriteFile(path, []byte("older"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(release(files), path, "0.3.0", Stable, &out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "#!/bin/sh\nnewer\n" {
		t.Fatalf("binary is %q", got)
	}
	if !strings.Contains(out.String(), "0.3.0 -> 0.4.0") {
		t.Fatalf("said %q", out.String())
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o755 {
		t.Fatalf("mode is %v, an update must stay executable", fi.Mode().Perm())
	}
}

// The point of the whole package: a substituted download never reaches the
// path the user runs, and what was there before is still there.
func TestRunLeavesTheOldBinaryWhenTheChecksumIsWrong(t *testing.T) {
	files, asset := fleet(t, "v0.4.0", "good")
	files[downloadURL+"/v0.4.0/"+asset] = "tampered"
	path := filepath.Join(t.TempDir(), "siltide")
	if err := os.WriteFile(path, []byte("older"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Run(release(files), path, "0.3.0", Stable, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error is %v, want a checksum mismatch", err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "older" {
		t.Fatalf("binary is %q, a failed update must change nothing", b)
	}
	// And no debris was left behind next to it.
	ents, _ := os.ReadDir(filepath.Dir(path))
	if len(ents) != 1 {
		t.Fatalf("%d files left in the directory", len(ents))
	}
}

func TestRunRefusesAReleaseWithNoChecksumForThisAsset(t *testing.T) {
	files, _ := fleet(t, "v0.4.0", "good")
	files[downloadURL+"/v0.4.0/checksums.txt"] = sum("x") + "  siltide-plan9-386\n"
	path := filepath.Join(t.TempDir(), "siltide")
	if err := os.WriteFile(path, []byte("older"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Run(release(files), path, "0.3.0", Stable, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no checksum") {
		t.Fatalf("error is %v", err)
	}
}

func TestRunSaysNothingToDoOnTheNewestVersion(t *testing.T) {
	files, _ := fleet(t, "v0.4.0", "body")
	var out bytes.Buffer
	if err := Run(release(files), filepath.Join(t.TempDir(), "siltide"), "v0.4.0", Stable, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "is the newest") {
		t.Fatalf("said %q", out.String())
	}
}

func TestRunRefusesAPackagedInstall(t *testing.T) {
	for path, by := range map[string]string{
		"/usr/bin/siltide": "your package manager",
		"/nix/store/abc123-siltide-0.3.0/bin/siltide":    "nix",
		"/opt/homebrew/Cellar/siltide/0.3.0/bin/siltide": "brew",
	} {
		err := Run(release(nil), path, "0.3.0", Stable, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), by) {
			t.Fatalf("%s: error is %v, want it to name %s", path, err, by)
		}
	}
}

// Where the install script puts a binary is exactly where updating has to work.
func TestManagedLeavesTheInstallScriptsTargetsAlone(t *testing.T) {
	for _, path := range []string{
		"/usr/local/bin/siltide",
		"/home/ada/.local/bin/siltide",
		"/Users/ada/bin/siltide",
	} {
		if by, ok := Managed(path); ok {
			t.Fatalf("%s reported as managed by %s", path, by)
		}
	}
}

func TestChannelOfFollowsTheRunningBuild(t *testing.T) {
	for v, want := range map[string]Channel{
		"0.3.0":         Stable,
		"v0.3.0":        Stable,
		"0.4.0-rc.1":    Dev,
		"v0.4.0-dev.12": Dev,
		"0.4.0-next":    Dev,
	} {
		if got := ChannelOf(v); got != want {
			t.Errorf("ChannelOf(%q) = %v, want %v", v, got, want)
		}
	}
}

func TestLatestOnDevTakesTheNewestNonDraft(t *testing.T) {
	get := release(map[string]string{
		releasesURL: `[{"tag_name": "v0.5.0-rc.2", "draft": true},
		               {"tag_name": "v0.5.0-rc.1", "prerelease": true},
		               {"tag_name": "v0.4.0"}]`,
	})
	r, err := Latest(get, Dev)
	if err != nil {
		t.Fatal(err)
	}
	if r.Version() != "0.5.0-rc.1" {
		t.Fatalf("chose %q, a draft is not published yet", r.Version())
	}
}

func TestLatestReportsAnEmptyReleaseList(t *testing.T) {
	if _, err := Latest(release(map[string]string{releasesURL: `[]`}), Dev); err == nil {
		t.Fatal("want an error when there is nothing to move to")
	}
	if _, err := Latest(release(map[string]string{latestURL: `{}`}), Stable); err == nil {
		t.Fatal("want an error when the API names no version")
	}
}

func TestChecksumReadsTheGoreleaserFormat(t *testing.T) {
	sums := []byte(strings.Join([]string{
		sum("a") + "  siltide-linux-amd64",
		sum("b") + "  siltide-darwin-arm64",
		"not a checksum line",
		"deadbeef  siltide-short-sum", // too short to be SHA-256
	}, "\n"))
	if got, ok := Checksum(sums, "siltide-darwin-arm64"); !ok || got != sum("b") {
		t.Fatalf("got %q %v", got, ok)
	}
	if _, ok := Checksum(sums, "siltide-short-sum"); ok {
		t.Fatal("a truncated digest is not a checksum")
	}
	if _, ok := Checksum(sums, "siltide-windows-amd64"); ok {
		t.Fatal("want no checksum for an asset that is not published")
	}
}

func TestPrepareNamesTheAssetForThisMachine(t *testing.T) {
	files, asset := fleet(t, "v0.4.0", "body")
	p, err := Prepare(release(files), "v0.3.0", Stable)
	if err != nil {
		t.Fatal(err)
	}
	if p.From != "0.3.0" || p.To != "0.4.0" || !p.Newer() {
		t.Fatalf("plan is %+v", p)
	}
	if !strings.HasSuffix(p.URL, "/v0.4.0/"+asset) {
		t.Fatalf("url is %s", p.URL)
	}
	if !strings.Contains(asset, runtime.GOOS) || !strings.Contains(asset, runtime.GOARCH) {
		t.Fatalf("asset %s does not name this machine", asset)
	}
}

func TestPrepareReportsAFetchFailure(t *testing.T) {
	want := errors.New("no network")
	_, err := Prepare(func(string) ([]byte, error) { return nil, want }, "0.3.0", Stable)
	if !errors.Is(err, want) {
		t.Fatalf("error is %v", err)
	}
}

func TestInstallReplacesAFileInUse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "siltide")
	if err := os.WriteFile(path, []byte("older"), 0o700); err != nil {
		t.Fatal(err)
	}
	// Hold the old file open, as the running process does on Unix: the rename
	// has to succeed anyway.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := Install(path, []byte("newer")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "newer" {
		t.Fatalf("binary is %q", b)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0o700 {
		t.Fatalf("mode is %v, want the mode of what it replaced", fi.Mode().Perm())
	}
}

func TestInstallReportsADirectoryItCannotWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o700) }()
	err := Install(filepath.Join(dir, "siltide"), []byte("newer"))
	if err == nil || !strings.Contains(err.Error(), "cannot write to") {
		t.Fatalf("error is %v", err)
	}
}

func TestVerify(t *testing.T) {
	if err := Verify([]byte("body"), sum("body")); err != nil {
		t.Fatal(err)
	}
	if err := Verify([]byte("body"), sum("other")); err == nil {
		t.Fatal("want a mismatch")
	}
}

func TestSelfIsAFileThatExists(t *testing.T) {
	path, err := Self()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

// Before the first release exists, both endpoints answer 404. That is not a
// failure to report as one.
func TestLatestReadsA404AsNothingPublished(t *testing.T) {
	get := func(url string) ([]byte, error) {
		return nil, &StatusError{URL: url, Status: "404 Not Found", Code: 404}
	}
	for _, ch := range []Channel{Stable, Dev} {
		if _, err := Latest(get, ch); !errors.Is(err, ErrNoRelease) {
			t.Fatalf("channel %v: error is %v, want ErrNoRelease", ch, err)
		}
	}
}

func TestLatestKeepsOtherFailures(t *testing.T) {
	get := func(url string) ([]byte, error) {
		return nil, &StatusError{URL: url, Status: "503 Service Unavailable", Code: 503}
	}
	_, err := Latest(get, Stable)
	if errors.Is(err, ErrNoRelease) || err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("error is %v, a server fault is not an empty release list", err)
	}
}

func TestRunNamesTheChannelItStayedOn(t *testing.T) {
	files, _ := fleet(t, "v0.4.0-rc.1", "body")
	var out bytes.Buffer
	if err := Run(release(files), filepath.Join(t.TempDir(), "siltide"), "0.4.0-rc.1", ChannelOf("0.4.0-rc.1"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "dev channel") {
		t.Fatalf("said %q", out.String())
	}
}
