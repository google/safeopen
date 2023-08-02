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
	"fmt"
	"io"
	"os"
	"path"
	"testing"
)

func TestAtOpenat2(t *testing.T) {
	if !isOpenat2WithResolveBeneathSupported() {
		t.Skip()
		return
	}

	expectedData := "hello"

	tmpDir := t.TempDir()
	dataFile := "some.txt"
	fullDataFile := path.Join(tmpDir, dataFile)
	err := os.WriteFile(fullDataFile, []byte(expectedData), 0644)
	if err != nil {
		t.Fatalf("os.WriteFile(%q) error: %v", fullDataFile, err)
	}

	legitLinkFile := "legit.link"
	fullLegitLinkFile := path.Join(tmpDir, legitLinkFile)
	err = os.Symlink(dataFile, fullLegitLinkFile)
	if err != nil {
		t.Fatalf("os.Symlink(%q, %q) error: %v", dataFile, fullLegitLinkFile, err)
	}

	relLinkFile := "relative.traverse.link"
	fullRelLinkFile := path.Join(tmpDir, relLinkFile)
	err = os.Symlink("../../../etc/passwd", fullRelLinkFile)
	if err != nil {
		t.Fatalf(`os.Symlink("../../../etc/passwd", %q) error: %v`, fullRelLinkFile, err)
	}

	absLinkFile := "absolute.traverse.link"
	fullAbsLinkFile := path.Join(tmpDir, absLinkFile)
	err = os.Symlink("/etc/passwd", fullAbsLinkFile)
	if err != nil {
		t.Fatalf(`os.Symlink("/etc/passwd", %q) error: %v`, fullAbsLinkFile, err)
	}

	// We are on linux with openat2 available and following a relative symbolic link which resolves
	// beneath the basedir. Still, OpenAt is documented to open files directly within the basedir,
	// so all symlinks are rejected here - regardless openat2 support.
	_, err = OpenAt(tmpDir, legitLinkFile)
	if err == nil {
		t.Fatalf("OpenAt(%q, %q) should have been an error", tmpDir, legitLinkFile)
	}

	f, err := OpenBeneath(tmpDir, legitLinkFile)
	if err != nil {
		t.Fatalf("OpenBeneath(%q, %q) error: %v", tmpDir, legitLinkFile, err)
	}
	defer f.Close()
	d, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("io.ReadAll() error: %v", err)
	}
	if string(d) != expectedData {
		t.Errorf("read %q, want %q", string(d), expectedData)
	}

	for _, s := range []string{relLinkFile, absLinkFile} {
		_, err = OpenAt(tmpDir, s)
		if err == nil {
			t.Errorf("OpenAt(%q, %q) should have been an error", tmpDir, s)
		}
		_, err = OpenBeneath(tmpDir, s)
		if err == nil {
			t.Errorf("OpenBeneath(%q, %q) should have been an error", tmpDir, s)
		}
	}
}

func getNumberOfFds() (int, error) {
	files, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0, err
	}
	return len(files), nil
}

func TestLinuxDirTraversal(t *testing.T) {
	origForceLegacyMode := forceLegacyMode
	defer func() { forceLegacyMode = origForceLegacyMode }()

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

	for i := 0; i < 2; i++ {
		forceLegacyMode = i != 0
		t.Run(fmt.Sprintf("LegacyMode%d", i), func(t *testing.T) {

			fdsBefore, err := getNumberOfFds()
			if err != nil {
				t.Fatalf("unable to query number of file descriptors before the OpenBeneath call: %v", err)
			}

			f, err := OpenBeneath(tmpdir, dataFile)
			if err != nil {
				t.Fatalf("OpenBeneath(%q, %q) error: %v", tmpdir, dataFile, err)
			}
			actualData, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				t.Fatalf("io.ReadAll() error: %v", err)
			}

			if string(actualData) != fileContent {
				t.Errorf("io.ReadAll() = %v, want = %v", string(actualData), fileContent)
			}
			fdsAfter, err := getNumberOfFds()
			if err != nil {
				t.Fatalf("unable to query number of file descriptors after the OpenBeneath call: %v", err)
			}

			if fdsBefore != fdsAfter {
				t.Errorf("OpenBeneath() leaked file descriptors (before = %v, after = %v)", fdsBefore, fdsAfter)
			}

		})
	}
}
