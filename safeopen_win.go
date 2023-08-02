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

//go:build windows
// +build windows

package safeopen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func winIsSimpleFilename(path string) bool {
	return !(strings.Contains(path, "/") || strings.Contains(path, `\`) || path == "." || path == "..")
}

func winRelativePathDoesntTraverse(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	// On Windows, ? is an invalid filename character it may be present in special paths only
	// (e.g.: \??\...)
	if strings.Contains(path, "?") {
		return "", false
	}

	hasDots := false
	// Normalizing directory separator as Windows accepts both:
	path = strings.ReplaceAll(path, `/`, `\`)
	for p := path; p != ""; {
		var part string
		part, p, _ = strings.Cut(p, `\`)
		if part == "." || part == ".." {
			hasDots = true
			break
		}
	}
	if hasDots {
		path = filepath.Clean(path)
	}
	if path == ".." || strings.HasPrefix(path, `..\`) {
		return "", false
	}
	return path, true
}

func winOpenAt(dfd windows.Handle, file string, access, disposition, options uint32) (windows.Handle, error) {
	var allocSize int64 = 0
	var iosb windows.IO_STATUS_BLOCK

	objectName, err := windows.NewNTUnicodeString(file)
	if err != nil {
		return windows.InvalidHandle, err
	}
	oa := &windows.OBJECT_ATTRIBUTES{
		ObjectName: objectName,
	}
	if dfd != windows.InvalidHandle {
		oa.RootDirectory = dfd
	}
	oa.Length = uint32(unsafe.Sizeof(*oa))

	var fileHandle windows.Handle
	err = windows.NtCreateFile(&fileHandle,
		access,
		oa, &iosb, &allocSize, windows.FILE_ATTRIBUTE_NORMAL,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		disposition,
		options|windows.FILE_OPEN_REPARSE_POINT,
		0, 0)
	return fileHandle, err
}

func openFileAt(directory, file string, flag int, perm os.FileMode) (*os.File, error) {
	if !winIsSimpleFilename(file) {
		return nil, &os.PathError{"OpenAt", file, errors.New("invalid filename")}
	}

	return openFileBeneath(directory, file, flag, perm)
}

func openFileBeneath(directory, file string, flag int, _ os.FileMode) (*os.File, error) {
	var winPerm uint32 = windows.FILE_GENERIC_READ
	if flag != os.O_RDONLY {
		winPerm |= windows.FILE_GENERIC_WRITE
	}

	// Note, on Windows the semantics of disposition options are different compared to posix,
	// os.O_CREATE|os.O_TRUNC => FILE_CREATE|FILE_OVERWRITE is invalid
	var disposition uint32 = windows.FILE_OPEN
	if flag&os.O_CREATE > 0 {
		disposition = windows.FILE_CREATE
	}
	if flag&os.O_TRUNC > 0 {
		disposition = windows.FILE_OVERWRITE_IF
	}

	sanitizedFile, safe := winRelativePathDoesntTraverse(file)
	if !safe {
		return nil, &os.PathError{"OpenAt", file, errors.New("invalid filename")}
	}

	dfd, err := winOpenAt(windows.InvalidHandle, `\??\`+directory,
		winPerm,
		windows.FILE_OPEN,
		windows.FILE_DIRECTORY_FILE)
	if err != nil {
		return nil, err
	}

	segs := strings.Split(sanitizedFile, `\`)

	if len(segs) > 1 {
		for _, seg := range segs[:len(segs)-1] {
			// Ignore empty segments
			if seg == "" {
				continue
			}

			odfd := dfd
			dfd, err = winOpenAt(dfd, seg, winPerm, windows.FILE_OPEN, windows.FILE_DIRECTORY_FILE)
			windows.CloseHandle(odfd)

			if err != nil {
				return nil, err
			}
		}
	}

	// Note: windows.FILE_SYNCHRONOUS_IO_NONALERT is important here, without that regular file IO
	// would be rejected with the error message "The parameter is incorrect".
	fd, err := winOpenAt(dfd, segs[len(segs)-1], winPerm, disposition,
		windows.FILE_RANDOM_ACCESS|windows.FILE_NON_DIRECTORY_FILE|windows.FILE_SYNCHRONOUS_IO_NONALERT)
	windows.CloseHandle(dfd)

	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), filepath.Join(directory, sanitizedFile)), nil
}
