package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func removeStoredFileWithinRoot(path, root string) error {
	path = strings.TrimSpace(path)
	root = strings.TrimSpace(root)
	if path == "" {
		return nil
	}
	if root == "" {
		return fmt.Errorf("storage root is empty")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(absoluteRoot, absolutePath)
	if err != nil {
		return err
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing to delete path outside storage root")
	}

	info, err := os.Lstat(absolutePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("refusing to delete a directory")
	}
	if err := os.Remove(absolutePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
