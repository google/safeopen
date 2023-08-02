// Copyright 2024 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package safeopen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

var (
	forceLegacyMode bool
)

func canTraverseUnixRelPath(path string) (string, bool) {
	if path == "" {
		return "", false
	}

	// openat2 returns "invalid cross-device link" for absolute destinations,
	// and we want to keep backward compatibility
	path = strings.TrimLeft(path, "/")

	hasDots := false
	for p := path; p != ""; {
		var part string
		part, p, _ = strings.Cut(p, "/")
		if part == "." || part == ".." {
			hasDots = true
			break
		}
	}
	if hasDots {
		path = filepath.Clean(path)
	}
	if path == ".." || strings.HasPrefix(path, "../") {
		return "", false
	}

	return path, true
}

func openFileAt(directory, file string, flag int, perm os.FileMode) (*os.File, error) {
	if !unixIsFilename(file) {
		return nil, &os.PathError{"OpenAt", file, errors.New("invalid filename")}
	}

	return openFileImpl(directory, file, flag, perm, unix.RESOLVE_NO_SYMLINKS)
}

func openFileBeneath(directory, file string, flag int, perm os.FileMode) (*os.File, error) {
	file, safe := canTraverseUnixRelPath(file)
	if !safe {
		return nil, &os.PathError{"OpenBeneath", file, errors.New("invalid filename")}
	}

	return openFileImpl(directory, file, flag, perm, 0)
}

func openFileImpl(directory, file string, flag int, perm os.FileMode, resolveHow uint64) (*os.File, error) {
	dfd, err := unix.Open(directory, os.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	defer unix.Close(dfd)

	fd, err := openFileImplBeneathFirst(dfd, file, flag, perm, resolveHow)
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), filepath.Join(directory, file)), nil
}

func openFileImplBeneathFirst(dfd int, file string, flag int, perm os.FileMode, resolveHow uint64) (int, error) {
	if forceLegacyMode {
		return openFileImplLegacy(dfd, file, flag, perm)
	}

	fd, supported, err := openFileImplBeneath(dfd, file, flag, perm, resolveHow)
	if !supported {
		return openFileImplLegacy(dfd, file, flag, perm)
	}
	return fd, err
}

func openFileImplBeneath(dfd int, file string, flag int, perm os.FileMode, resolveHow uint64) (int, bool, error) {
	fd, err := unix.Openat2(dfd, file, &unix.OpenHow{
		Flags:   uint64(flag),
		Mode:    uint64(syscallMode(perm)),
		Resolve: unix.RESOLVE_BENEATH | resolveHow,
	})
	supported := true
	if err != nil {
		// If openat2 is not available at all, ENOSYS is returned.
		// According to the docs, if RESOLVE_BENEATH is not supported, EINVAL is returned.
		// errors.ErrUnsupported seems to be in go1.21 only, which is not yet available for bazel
		// (safeopen_linux.go:116:61: undefined: errors.ErrUnsupported)
		if errs, ok := err.(syscall.Errno); ok && (errs == syscall.ENOSYS || errs == syscall.ENOTSUP || errs == syscall.EOPNOTSUPP || errs == syscall.EINVAL) {
			// Falling back to legacy impl.
			supported = false
		}
		return 0, supported, err
	}
	return fd, supported, nil
}

func openFileImplLegacy(dfd int, file string, flag int, perm os.FileMode) (int, error) {
	segs := strings.Split(file, string(filepath.Separator))

	adfd := dfd
	var err error
	if len(segs) > 1 {
		for _, seg := range segs[:len(segs)-1] {
			// Ignore empty segments
			if seg == "" {
				continue
			}

			odfd := adfd

			adfd, err = unix.Openat(adfd, seg, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)

			// odfd (the previous adfd) is not needed any longer. Closing it right now.
			if odfd != dfd {
				if cerr := unix.Close(odfd); cerr != nil {
					return 0, cerr
				}
			}

			if err != nil {
				return 0, err
			}

		}
	}

	fd, err := unix.Openat(adfd, segs[len(segs)-1], flag|syscall.O_NOFOLLOW, syscallMode(perm))
	if adfd != dfd {
		err = unix.Close(adfd)
	}
	return fd, err
}

// isOpenat2WithResolveBeneathSupported is a helper function for unit tests only.
func isOpenat2WithResolveBeneathSupported() bool {
	dfd, err := unix.Open("/etc", os.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return false
	}
	defer unix.Close(dfd)

	fd, supported, err := openFileImplBeneath(dfd, "passwd", os.O_RDONLY, 0, 0)
	if err != nil {
		return false
	}
	unix.Close(fd)
	return supported
}
