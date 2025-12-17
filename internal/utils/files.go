package utils

import (
	"io"
	"os"
)

func FileCopy(src, dst string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	if err := os.Chmod(dst, 0700); err != nil {
		return err
	}

	return dstFile.Close()
}
