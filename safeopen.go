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

// Package safeopen provides replacement APIs for Open that do not permit path traversal.
// The library supports Unix and Windows systems. OS native safe primitives are leveraged where
// available (e.g. openat2 + RESOLVE_BENEATH).
// Symbolic links are followed only if there is a safe way to prevent traversal (e.g. on platforms
// where OS level safe primitives are available), otherwise an error is returned.
package safeopen

import (
	"io"
	"os"
)

// OpenAt opens the named file in the named directory for reading.
// file may not contain path separators.
//
// If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func OpenAt(directory, file string) (*os.File, error) {
	return OpenFileAt(directory, file, os.O_RDONLY, 0)
}

// CreateAt creates or truncates the named file in the named directory.
// file may not contain path separators.
//
// If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func CreateAt(directory, file string) (*os.File, error) {
	return OpenFileAt(directory, file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// OpenFileAt is the generalized OpenAt call; most users will use OpenAt
// or CreateAt instead.
//
// It opens the named file in the named directory with specified flag
// (O_RDONLY etc.). File may not contain path separators. If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). The perm parameter is ignored on Windows.
// If successful, methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func OpenFileAt(directory, file string, flag int, perm os.FileMode) (*os.File, error) {
	return openFileAt(directory, file, flag, perm)
}

// OpenBeneath opens the named file in the named directory, or a subdirectory, for reading.
// file may not contain .. path traversal entries.
//
// If successful, methods on
// the returned file can be used for reading; the associated file
// descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func OpenBeneath(directory, file string) (*os.File, error) {
	return OpenFileBeneath(directory, file, os.O_RDONLY, 0)
}

// CreateBeneath creates or truncates the named file in the named directory.
// file may not contain .. path traversal entries.
//
// If the file already exists,
// it is truncated. If the file does not exist, it is created with mode 0666
// (before umask). If successful, methods on the returned File can
// be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func CreateBeneath(directory, file string) (*os.File, error) {
	return OpenFileBeneath(directory, file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// OpenFileBeneath is the generalized OpenBeneath call; most users will use OpenBeneath
// or CreateBeneath instead.
//
// It opens the named file in the named directory with specified flag
// (O_RDONLY etc.). File may not contain .. path traversal entries.
// If the file does not exist, and the O_CREATE flag
// is passed, it is created with mode perm (before umask). The perm parameter is ignored on Windows.
// If successful, methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func OpenFileBeneath(directory, file string, flag int, perm os.FileMode) (*os.File, error) {
	return openFileBeneath(directory, file, flag, perm)
}

type openerFunc func(dir, file string, flag int, perm os.FileMode) (*os.File, error)

func readFile(directory, file string, opener openerFunc) ([]byte, error) {
	f, err := opener(directory, file, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func writeFile(directory, file string, data []byte, perm os.FileMode, creator openerFunc) error {
	f, err := creator(directory, file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

// ReadFileAt is a replacement of os.ReadFile that leverages safeopen.OpenAt.
func ReadFileAt(directory, file string) ([]byte, error) {
	return readFile(directory, file, OpenFileAt)
}

// WriteFileAt is a replacement of os.WriteFile that leverages safeopen.CreateAt.
func WriteFileAt(directory, file string, data []byte, perm os.FileMode) error {
	return writeFile(directory, file, data, perm, OpenFileAt)
}

// ReadFileBeneath is a replacement of os.ReadFile that leverages safeopen.OpenBeneath.
func ReadFileBeneath(directory, file string) ([]byte, error) {
	return readFile(directory, file, OpenFileBeneath)
}

// WriteFileBeneath is a replacement of os.WriteFile that leverages safeopen.CreateBeneath.
func WriteFileBeneath(directory, file string, data []byte, perm os.FileMode) error {
	return writeFile(directory, file, data, perm, OpenFileBeneath)
}
