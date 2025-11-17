package findCsv

import (
	"os"
	"path/filepath"
	"strings"
)

func FindAnyCSV(rootDir string) (string, error) {
	dirEntries, err := os.ReadDir(rootDir)
	if err != nil {
		return "", err
	}
	for _, entry := range dirEntries {
		if entry.IsDir() {
			folderPath := filepath.Join(rootDir, entry.Name())
			files, err := os.ReadDir(folderPath)
			if err != nil {
				return "", err
			}
			for _, subentry := range files {
				if !subentry.IsDir() && strings.HasSuffix(subentry.Name(), ".csv") {

					return filepath.Join(folderPath, subentry.Name()), nil
				}
			}
		} else if strings.HasSuffix(entry.Name(), ".csv") {
			return filepath.Join(rootDir, entry.Name()), nil
		}
	}
	return "", os.ErrNotExist
}
