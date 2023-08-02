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

//go:build unix
// +build unix

package safeopen

import (
	"io"
	"os"
	"path"

	"testing"
)

func prepareUnixStructure(t *testing.T) string {
	t.Helper()
	tmpdir := t.TempDir()

	// Prepare file structure for subsequent tests
	f1, err := os.OpenFile(path.Join(tmpdir, "safeopentarget"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()

	err = os.Mkdir(path.Join(tmpdir, "subdir"), 0777)
	if err != nil {
		t.Fatal(err)
	}

	f2, err := os.OpenFile(path.Join(tmpdir, "subdir", "safeopentarget"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	err = os.Symlink(path.Join(tmpdir, "subdir"), path.Join(tmpdir, "safeopensym"))
	if err != nil {
		t.Fatal(err)
	}

	return tmpdir
}

func TestUnixSafeOpenAt(t *testing.T) {
	basedir := prepareUnixStructure(t)

	type testCase struct {
		basedir       string
		file          string
		expectedError string
	}
	testCases := []testCase{
		{basedir, "safeopentarget", ""},
		{basedir + "/", "safeopentarget", ""},
		{basedir, "subdir/safeopentarget", "subdirectory"},
		{path.Join(basedir, "subdir"), "../safeopentarget", "traversal"},
		{basedir, "subdir/../safeopentarget", "traversal"},
		{basedir, "safeopensym", "symlink"},
	}
	for _, tc := range testCases {
		fd, err := OpenAt(tc.basedir, tc.file)
		t.Logf("OpenAt(%q, %q) => %v", tc.basedir, tc.file, err)
		if err == nil && fd == nil {
			t.Errorf("OpenAt(%q, %q) did not throw an error, but did not return a file handle", tc.basedir, tc.file)
		}
		if err != nil && fd != nil {
			t.Errorf("OpenAt(%q, %q) did throw an error, but also returned a file handle", tc.basedir, tc.file)
		}

		if err != nil && tc.expectedError == "" {
			t.Errorf("OpenAt(%q, %q) returned an error = %v", tc.basedir, tc.file, err)
		} else if err == nil && tc.expectedError != "" {
			t.Fatalf("OpenAt(%q, %q) expected error for %v", tc.basedir, tc.file, tc.expectedError)
		}

		if fd != nil {
			fd.Close()
		}
	}
}

func TestUnixSafeOpenBeneath(t *testing.T) {
	basedir := prepareUnixStructure(t)

	type testCase struct {
		basedir       string
		file          string
		expectedError string
	}
	testCases := []testCase{
		{basedir, "safeopentarget", ""},
		{basedir, "subdir/safeopentarget", ""},
		{basedir, "/subdir/safeopentarget", ""},
		{path.Join(basedir, "subdir"), "../safeopentarget", "traversal"},
		{basedir, "subdir/../safeopentarget", ""},
		{basedir, "safeopensym", "symlink"},
		{basedir, "safeopensym/safeopentarget", "symlink"},
	}

	for _, tc := range testCases {
		fd, err := OpenBeneath(tc.basedir, tc.file)
		t.Logf("OpenBeneath(%q, %q) => %v", tc.basedir, tc.file, err)
		if err == nil && fd == nil {
			t.Errorf("OpenBeneath(%q, %q) did not throw an error, but did not return a file handle", tc.basedir, tc.file)
		}
		if err != nil && fd != nil {
			t.Errorf("OpenBeneath(%q, %q) did throw an error, but also returned a file handle", tc.basedir, tc.file)
		}

		if err != nil && tc.expectedError == "" {
			t.Errorf("OpenBeneath(%q, %q) returned an error = %v", tc.basedir, tc.file, err)
		} else if err == nil && tc.expectedError != "" {
			t.Fatalf("OpenBeneath(%q, %q) expected error for %v", tc.basedir, tc.file, tc.expectedError)
		}

		if fd != nil {
			fd.Close()
		}
	}
}

func TestUnixDirTraversal(t *testing.T) {
	tmpdir := t.TempDir()

	err := os.Mkdir(path.Join(tmpdir, "subdir"), 0777)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Mkdir(path.Join(tmpdir, "subdir", "subsubdir"), 0777)
	if err != nil {
		t.Fatal(err)
	}

	fileContent := "hello"
	err = os.WriteFile(path.Join(tmpdir, "subdir", "subsubdir", "data.txt"), []byte(fileContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	dataFile := path.Join("subdir", "subsubdir", "data.txt")
	f, err := OpenBeneath(tmpdir, dataFile)
	if err != nil {
		t.Fatalf("OpenBeneath(%q, %q) error: %v", tmpdir, dataFile, err)
	}
	defer f.Close()
	actualData, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("io.ReadAll() error: %v", err)
	}

	if string(actualData) != fileContent {
		t.Errorf("io.ReadAll() = %v, want = %v", string(actualData), fileContent)
	}
}
