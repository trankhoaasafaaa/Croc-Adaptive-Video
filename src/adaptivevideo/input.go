package adaptivevideo

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const MaxDeadline = 24 * time.Hour

type Source struct {
	Variant
	Path string
	info os.FileInfo
}

type Request struct {
	Deadline time.Duration
	Sources  []Source
}

func Parse(deadline time.Duration, specs []string) (*Request, error) {
	if deadline <= 0 || deadline > MaxDeadline {
		return nil, fmt.Errorf("adaptive video: --video-deadline must be greater than 0 and at most 24h")
	}
	if len(specs) < 2 || len(specs) > MaxVariants {
		return nil, fmt.Errorf("adaptive video: provide 2–16 --video-variant values; use croc send FILE for one file")
	}
	request := &Request{Deadline: deadline}
	seen := make(map[int]bool, len(specs))
	for _, spec := range specs {
		label, file, ok := strings.Cut(spec, "=")
		height, err := strconv.Atoi(label)
		if !ok || file == "" || err != nil || height <= 0 || height > MaxHeight || seen[height] {
			return nil, fmt.Errorf("adaptive video: expected unique HEIGHT=PATH with height between 1 and %d", MaxHeight)
		}
		if !strings.EqualFold(filepath.Ext(file), ".mp4") {
			return nil, fmt.Errorf("adaptive video: %q must have an .mp4 extension", file)
		}
		info, err := os.Lstat(file)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return nil, fmt.Errorf("adaptive video: %q must be a nonempty regular file; symlinks are unsupported", file)
		}
		for _, other := range request.Sources {
			if os.SameFile(info, other.info) {
				return nil, fmt.Errorf("adaptive video: repeated source file %q", file)
			}
		}
		source := Source{Variant: Variant{Height: height, Bytes: info.Size()}, Path: file, info: info}
		if err := source.checkMP4(); err != nil {
			return nil, fmt.Errorf("adaptive video: %q: %w", file, err)
		}
		seen[height] = true
		request.Sources = append(request.Sources, source)
	}
	return request, nil
}

// CheckUnchanged checks that selection still refers to the measured source.
func (s Source) CheckUnchanged() error {
	info, err := os.Lstat(s.Path)
	if err != nil {
		return err
	}
	if !s.matches(info) {
		return fmt.Errorf("adaptive video: source changed after validation: %q", s.Path)
	}
	return nil
}

func (s Source) matches(info os.FileInfo) bool {
	return s.info != nil && info.Mode().IsRegular() && info.Size() == s.Bytes &&
		info.ModTime().Equal(s.info.ModTime()) && os.SameFile(s.info, info)
}

// checkMP4 recognizes a leading ISO BMFF file-type box with an MP4 family
// brand. This is an envelope check, not a video-stream or codec validator.
func (s Source) checkMP4() error {
	file, err := os.Open(s.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !s.matches(info) {
		return fmt.Errorf("file changed during validation")
	}
	var header [8]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		return err
	}
	size := int64(binary.BigEndian.Uint32(header[:4]))
	if string(header[4:]) != "ftyp" || size < 16 || size > min(s.Bytes, 4096) || size%4 != 0 {
		return fmt.Errorf("expected a leading MP4 file-type box")
	}
	body := make([]byte, size-8)
	if _, err := io.ReadFull(file, body); err != nil {
		return err
	}
	for i := 0; i < len(body); i += 4 {
		if i == 4 { // The minor version is not a brand.
			continue
		}
		switch string(body[i : i+4]) {
		case "isom", "iso2", "iso3", "iso4", "iso5", "iso6", "mp41", "mp42", "avc1", "dash", "M4V ", "MSNV":
			return nil
		}
	}
	return fmt.Errorf("unsupported MP4 file-type brand")
}
