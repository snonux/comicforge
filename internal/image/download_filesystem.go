package image

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

type secureOutputStore struct {
	options *DownloadOptions
}

func newSecureOutputStore(options *DownloadOptions) secureOutputStore {
	return secureOutputStore{options: options}
}

func (s secureOutputStore) openSecureOutputFile(outputPath string) (*os.File, int, string, error) {
	baseAbs, err := s.baseAbs()
	if err != nil {
		return nil, 0, "", err
	}

	targetAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("resolve output path: %w", err)
	}
	relPath, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return nil, 0, "", fmt.Errorf("relativize output path: %w", err)
	}
	if relPath == "." || relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return nil, 0, "", fmt.Errorf("path escapes base dir")
	}

	relPath = filepath.Clean(relPath)
	parentRel := filepath.Dir(relPath)
	finalName := filepath.Base(relPath)

	parentFD, err := openDirPathNoFollow(baseAbs, parentRel, s.createDirs())
	if err != nil {
		return nil, 0, "", err
	}

	flags := unix.O_WRONLY | unix.O_CREAT | unix.O_CLOEXEC | unix.O_NOFOLLOW
	if s.options != nil && s.options.OverwriteExisting {
		flags |= unix.O_TRUNC
	} else {
		flags |= unix.O_EXCL
	}

	fd, err := unix.Openat(parentFD, finalName, flags, 0o644)
	if err != nil {
		_ = unix.Close(parentFD)
		return nil, 0, "", fmt.Errorf("open output file: %w", err)
	}

	return os.NewFile(uintptr(fd), targetAbs), parentFD, finalName, nil
}

func (s secureOutputStore) writeSecureRelativeFile(outputPath, relPath string, content []byte, perm os.FileMode) error {
	file, parentFD, finalName, err := s.openSecureRelativeFile(outputPath, relPath, perm)
	if err != nil {
		return err
	}
	defer func() {
		_ = unix.Close(parentFD)
	}()
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Write(content); err != nil {
		_ = unix.Unlinkat(parentFD, finalName, 0)
		return fmt.Errorf("write attribution file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = unix.Unlinkat(parentFD, finalName, 0)
		return fmt.Errorf("sync attribution file: %w", err)
	}
	return nil
}

func (s secureOutputStore) openSecureRelativeFile(outputPath, relPath string, perm os.FileMode) (*os.File, int, string, error) {
	baseAbs, err := s.baseAbs()
	if err != nil {
		return nil, 0, "", err
	}

	targetAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("resolve output path: %w", err)
	}
	baseRelPath, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return nil, 0, "", fmt.Errorf("relativize output path: %w", err)
	}
	if baseRelPath == "." || baseRelPath == ".." || strings.HasPrefix(baseRelPath, ".."+string(os.PathSeparator)) {
		return nil, 0, "", fmt.Errorf("path escapes base dir")
	}

	parentRel := filepath.Dir(filepath.Clean(baseRelPath))
	parentFD, err := openDirPathNoFollow(baseAbs, parentRel, true)
	if err != nil {
		return nil, 0, "", err
	}

	flags := unix.O_WRONLY | unix.O_CREAT | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_TRUNC
	fd, err := unix.Openat(parentFD, relPath, flags, uint32(perm.Perm()))
	if err != nil {
		_ = unix.Close(parentFD)
		return nil, 0, "", fmt.Errorf("open attribution file: %w", err)
	}

	return os.NewFile(uintptr(fd), targetAbs), parentFD, relPath, nil
}

func (s secureOutputStore) baseAbs() (string, error) {
	baseDir := "./images"
	if s.options != nil && s.options.OutputDir != "" {
		baseDir = s.options.OutputDir
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve base dir: %w", err)
	}
	return baseAbs, nil
}

func (s secureOutputStore) createDirs() bool {
	return s.options != nil && s.options.CreateDir
}

func openDirPathNoFollow(baseAbs, relPath string, createDirs bool) (int, error) {
	dirFD, err := openPathDirNoFollow(baseAbs, createDirs)
	if err != nil {
		return 0, err
	}

	relPath = filepath.Clean(relPath)
	if relPath == "." {
		return dirFD, nil
	}

	parts := strings.Split(relPath, string(os.PathSeparator))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if createDirs {
			if err := unix.Mkdirat(dirFD, part, 0o755); err != nil && !errors.Is(err, unix.EEXIST) {
				_ = unix.Close(dirFD)
				return 0, fmt.Errorf("create dir %q: %w", part, err)
			}
		}

		nextFD, err := unix.Openat(dirFD, part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			_ = unix.Close(dirFD)
			return 0, fmt.Errorf("open dir %q: %w", part, err)
		}
		_ = unix.Close(dirFD)
		dirFD = nextFD
	}

	return dirFD, nil
}

func openPathDirNoFollow(absPath string, createDirs bool) (int, error) {
	absPath = filepath.Clean(absPath)
	if absPath == string(os.PathSeparator) {
		return unix.Open(absPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	}

	parentAbs := filepath.Dir(absPath)
	leaf := filepath.Base(absPath)
	parentFD, err := openPathDirNoFollow(parentAbs, createDirs)
	if err != nil {
		return 0, err
	}

	if createDirs {
		if err := unix.Mkdirat(parentFD, leaf, 0o755); err != nil && !errors.Is(err, unix.EEXIST) {
			_ = unix.Close(parentFD)
			return 0, fmt.Errorf("create dir %q: %w", leaf, err)
		}
	}

	dirFD, err := unix.Openat(parentFD, leaf, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	_ = unix.Close(parentFD)
	if err != nil {
		return 0, fmt.Errorf("open dir %q: %w", leaf, err)
	}

	return dirFD, nil
}
