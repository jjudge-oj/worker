package cache

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func untarGzReader(r io.Reader, dest string) error {
	const (
		maxFiles      = 200_000
		maxFileSize   = int64(1 << 30) // 1 GiB
		maxTotalBytes = int64(5 << 30) // 5 GiB
	)

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var files int
	var total int64

	cleanDest := filepath.Clean(dest) + string(os.PathSeparator)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		files++
		if files > maxFiles {
			return fmt.Errorf("too many files in archive")
		}

		if hdr.Size < 0 || hdr.Size > maxFileSize {
			return fmt.Errorf("file too large: %s", hdr.Name)
		}
		total += hdr.Size
		if total > maxTotalBytes {
			return fmt.Errorf("archive too large (uncompressed)")
		}

		target := filepath.Join(dest, hdr.Name)
		cleanTarget := filepath.Clean(target)

		if !strings.HasPrefix(cleanTarget, cleanDest) {
			return fmt.Errorf("illegal path: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanTarget, 0755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(f, tr, hdr.Size)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}

		default:
			// Skip symlinks/devices/etc.
		}
	}
	return nil
}
