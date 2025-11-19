package tools

import (
	"archive/tar"
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func Untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // конец архива
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Создаём директорию
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			// Создаём файл и копируем содержимое из архива
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func Unzip(r io.ReaderAt, size int64, dest string) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return err
	}

	for _, file := range zr.File {
		target := filepath.Join(dest, file.Name)

		if file.FileInfo().IsDir() {
			// Создаём директорию
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return err
			}
			continue
		}

		// Создаём родительскую директорию
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		// Открываем файл из архива
		rc, err := file.Open()
		if err != nil {
			return err
		}

		// Создаём файл и копируем содержимое
		f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(f, rc); err != nil {
			f.Close()
			rc.Close()
			return err
		}
		f.Close()
		rc.Close()
	}

	return nil
}
