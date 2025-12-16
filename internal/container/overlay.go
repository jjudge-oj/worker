package container

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Overlay struct {
	Base     string
	LowerDir string
	UpperDir string
	WorkDir  string
}

func NewOverlay(base, lower string) (*Overlay, error) {
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, err
	}

	mntFlags := unix.MS_NOSUID | unix.MS_NODEV
	if err := unix.Mount("tmpfs", base, "tmpfs", uintptr(mntFlags), "size=1G"); err != nil {
		return nil, err
	}

	ov := &Overlay{
		Base:     base,
		LowerDir: lower,
		UpperDir: filepath.Join(base, "upper"),
		WorkDir:  filepath.Join(base, "work"),
	}

	for _, dir := range []string{ov.UpperDir, ov.WorkDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	return ov, nil
}

func Cleanup(o *Overlay) error {
	var errs []error

	if err := unix.Unmount(o.Base, 0); err != nil {
		errs = append(errs, err)
	}

	for _, dir := range []string{o.UpperDir, o.WorkDir} {
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, err)
		}
	}

	if err := os.RemoveAll(o.Base); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}
