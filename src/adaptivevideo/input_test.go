package adaptivevideo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseMP4Sources(t *testing.T) {
	dir := t.TempDir()
	envelope := []byte("\x00\x00\x00\x14ftypisom\x00\x00\x00\x00mp42")
	first := filepath.Join(dir, "name = with spaces.mp4")
	second := filepath.Join(dir, "second.MP4")
	invalid := filepath.Join(dir, "fake.mp4")
	empty := filepath.Join(dir, "empty.mp4")
	for file, data := range map[string][]byte{first: envelope, second: envelope, invalid: []byte("not a video"), empty: nil} {
		if err := os.WriteFile(file, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	directory := filepath.Join(dir, "folder.mp4")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	valid := []string{"1080=" + first, "360=" + second}
	got, err := Parse(time.Second, valid)
	if err != nil || len(got.Sources) != 2 || got.Sources[0].Path != first || got.Sources[0].Bytes != int64(len(envelope)) {
		t.Fatalf("Parse = %+v, %v", got, err)
	}
	for _, tc := range []struct {
		name     string
		specs    []string
		deadline time.Duration
	}{
		{"zero deadline", valid, 0},
		{"long deadline", valid, MaxDeadline + time.Second},
		{"single", valid[:1], time.Second},
		{"duplicate height", []string{valid[0], "1080=" + second}, time.Second},
		{"duplicate file", []string{valid[0], "360=" + first}, time.Second},
		{"invalid MP4", []string{valid[0], "360=" + invalid}, time.Second},
		{"empty", []string{valid[0], "360=" + empty}, time.Second},
		{"missing", []string{valid[0], "360=" + filepath.Join(dir, "missing.mp4")}, time.Second},
		{"directory", []string{valid[0], "360=" + directory}, time.Second},
		{"invalid height", []string{valid[0], "bad=" + second}, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.deadline, tc.specs); err == nil {
				t.Fatal("invalid input was accepted")
			}
		})
	}
	link := filepath.Join(dir, "linked.mp4")
	if err := os.Symlink(second, link); err == nil {
		if _, err := Parse(time.Second, []string{valid[0], "360=" + link}); err == nil {
			t.Error("accepted a symlink")
		}
	}
	if err := os.WriteFile(first, append(envelope, 0), 0600); err != nil {
		t.Fatal(err)
	}
	if err := got.Sources[0].CheckUnchanged(); err == nil {
		t.Fatal("source modification was not detected")
	}
}
