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

//go:build windows
// +build windows

package safeopen

import (
	"fmt"
	"os"
	"path"

	"testing"
)

func prepareWinStructure(t *testing.T) string {
	t.Helper()
	tmpdir := t.TempDir()

	// Prepare file structure for subsequent tests
	fd1, err := os.OpenFile(path.Join(tmpdir, "safeopentarget"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer fd1.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Mkdir(path.Join(tmpdir, "subdir"), 0777)
	if err != nil {
		t.Fatal(err)
	}

	fd2, err := os.OpenFile(path.Join(tmpdir, "subdir", "safeopentarget"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer fd2.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Symlink(path.Join(tmpdir, "subdir"), path.Join(tmpdir, "safeopensym"))
	if err != nil {
		t.Fatal(err)
	}

	return tmpdir
}

func TestWinSafeopenAt(t *testing.T) {
	basedir := prepareWinStructure(t)

	type testCase struct {
		basedir       string
		file          string
		expectedError string
	}
	testCases := []testCase{
		{basedir, "safeopentarget", ""},
		{basedir + `\`, "safeopentarget", ""},
		{basedir, "subdir/safeopentarget", "subdirectory"},
		{path.Join(basedir, "subdir"), "../safeopentarget", "traversal"},
		{path.Join(basedir, "subdir"), `..\safeopentarget`, "traversal"},
		{basedir, "subdir/../safeopentarget", "traversal"},
		{basedir, `subdir\..\safeopentarget`, "traversal"},
		{basedir, "safeopensym", "symlink"},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("TestWinSafeopenAt%d", i), func(t *testing.T) {
			fd, err := OpenAt(tc.basedir, tc.file)
			if err == nil && fd == nil {
				t.Errorf("OpenAt(%q, %q) did not throw an error, but did not return a file handle", tc.basedir, tc.file)
			}
			if err != nil && fd != nil {
				t.Errorf("OpenAt(%q, %q) did throw an error, but also returned a file handle", tc.basedir, tc.file)
			}
			if fd != nil {
				defer fd.Close()
			}

			if err != nil && tc.expectedError == "" {
				t.Errorf("OpenAt(%q, %q) returned an error = %v", tc.basedir, tc.file, err)
			} else if err == nil && tc.expectedError != "" {
				t.Errorf("OpenAt(%q, %q) expected error for %v", tc.basedir, tc.file, tc.expectedError)
			}
		})

	}
}

func TestWinSafeopenBeneath(t *testing.T) {
	basedir := prepareWinStructure(t)

	type testCase struct {
		basedir       string
		file          string
		expectedError string
	}
	testCases := []testCase{
		{basedir, "safeopentarget", ""},
		{basedir, "subdir/safeopentarget", ""},
		{basedir, `subdir\safeopentarget`, ""},
		{basedir, "/subdir/safeopentarget", ""},
		{basedir, `\subdir\safeopentarget`, ""},
		{path.Join(basedir, "subdir"), "../safeopentarget", "traversal"},
		{path.Join(basedir, "subdir"), `..\safeopentarget`, "traversal"},
		{basedir, "subdir/../safeopentarget", ""},
		{basedir, `subdir\..\safeopentarget`, ""},
		{basedir, "safeopensym", "symlink"},
		{basedir, "safeopensym/safeopentarget", "symlink"},
		{basedir, `safeopensym\safeopentarget`, "symlink"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("TestWinSafeopenAt%d", i), func(t *testing.T) {
			fd, err := OpenBeneath(tc.basedir, tc.file)
			if err == nil && fd == nil {
				t.Errorf("OpenBeneath(%q, %q) did not throw an error, but did not return a file handle", tc.basedir, tc.file)
			}
			if err != nil && fd != nil {
				t.Errorf("OpenBeneath(%q, %q) did throw an error, but also returned a file handle", tc.basedir, tc.file)
			}
			if fd != nil {
				defer fd.Close()
			}

			if err != nil && tc.expectedError == "" {
				t.Errorf("OpenBeneath(%q, %q) returned an error = %v", tc.basedir, tc.file, err)
			} else if err == nil && tc.expectedError != "" {
				t.Errorf("OpenBeneath(%q, %q) expected error for %v", tc.basedir, tc.file, tc.expectedError)
			}
		})
	}
}

func TestWinIO(t *testing.T) {
	tmpdir := t.TempDir()

	filename := "data.txt"
	content := "content data"

	f1, err := CreateAt(tmpdir, filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f1.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	f1.Close()

	// Prepare file structure for subsequent tests
	dat, err := os.ReadFile(path.Join(tmpdir, filename))
	if err != nil {
		t.Fatal(err)
	}
	if string(dat) != content {
		t.Errorf("ReadFile() = %q, want %q", dat, content)
	}

	// And reading it back via safeopen as well
	f2, err := OpenAt(tmpdir, filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	b := make([]byte, 1000)
	n, err := f2.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(content) {
		t.Errorf("Read() = %d, want %d", n, len(content))
	}
	aRead := string(b[0:n])
	if aRead != content {
		t.Errorf("Read() = %q, want %q", aRead, content)
	}
}
