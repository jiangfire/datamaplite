package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	sourceDir = "web/dist"
	targetDir = "internal/webui/generated"
	keepFile  = ".keep"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	info, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("frontend build output not found at %q: run `pnpm --dir web build` first", sourceDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("frontend build output path %q is not a directory", sourceDir)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}
	if err := cleanTargetDir(targetDir); err != nil {
		return fmt.Errorf("clean target dir: %w", err)
	}

	if err := filepath.WalkDir(sourceDir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(sourceDir, current)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(targetDir, relPath)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		return copyFile(current, targetPath, entry)
	}); err != nil {
		return fmt.Errorf("copy frontend assets: %w", err)
	}

	return nil
}

func cleanTargetDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == keepFile {
			continue
		}

		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(sourcePath string, targetPath string, entry fs.DirEntry) (err error) {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sourceFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	info, err := entry.Info()
	if err != nil {
		return err
	}

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := targetFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}

	return nil
}
