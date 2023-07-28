// Copyright 2023 Google LLC
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

//go:build unix && !linux
// +build unix,!linux

package safeopen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func unixRelativePathDoesntTraverse(path string) bool {
	if path == "" {
		return false
	}
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
		return false
	}
	return true
}

func openFileAt(directory, file string, flag int, perm os.FileMode) (*os.File, error) {
	if !unixIsFilename(file) {
		return nil, &os.PathError{"OpenAt", file, errors.New("invalid filename")}
	}

	dfd, err := unix.Open(directory, os.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}

	fd, err := unix.Openat(dfd, file, flag|syscall.O_NOFOLLOW, syscallMode(perm))
	unix.Close(dfd)

	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), filepath.Join(directory, file)), nil
}

func openFileBeneath(directory, file string, flag int, perm os.FileMode) (*os.File, error) {
	if !unixRelativePathDoesntTraverse(file) {
		return nil, &os.PathError{"OpenBeneath", file, errors.New("invalid filename")}
	}

	dfd, err := unix.Open(directory, os.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}

	segs := strings.Split(file, string(filepath.Separator))

	if len(segs) > 1 {
		for _, seg := range segs[:len(segs)-1] {
			// Ignore empty segments
			if seg == "" {
				continue
			}

			odfd := dfd

			dfd, err = unix.Openat(dfd, seg, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
			unix.Close(odfd)

			if err != nil {
				return nil, err
			}
		}
	}

	fd, err := unix.Openat(dfd, segs[len(segs)-1], flag|syscall.O_NOFOLLOW, syscallMode(perm))
	unix.Close(dfd)

	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), filepath.Join(directory, file)), nil
}
