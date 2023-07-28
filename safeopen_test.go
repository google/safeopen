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

package safeopen

import (
	"os"
	"path"
	"testing"
)

func TestAt(t *testing.T) {
	filename := "something.txt"
	edata := []byte("content")

	tmpDir := t.TempDir()
	if err := WriteFileAt(tmpDir, filename, edata, 0644); err != nil {
		t.Fatal(err)
	}
	adata, err := ReadFileAt(tmpDir, filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(adata) != string(edata) {
		t.Errorf("ReadFileAt(%q, %q) = %q, want %q", tmpDir, filename, adata, edata)
	}

	if err = os.Mkdir(path.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	filenameInSubdir := path.Join("subdir", filename)
	if err = WriteFileAt(tmpDir, filenameInSubdir, edata, 0o0644); err == nil {
		t.Errorf("WriteFileAt(%q, %q) = %q want nil", tmpDir, filenameInSubdir, err)
	}

	if err = os.WriteFile(path.Join(tmpDir, filenameInSubdir), edata, 0644); err != nil {
		t.Fatal(err)
	}

	adata, err = ReadFileAt(tmpDir, filenameInSubdir)
	if adata != nil || err == nil {
		t.Errorf("ReadFileAt(%q, %q) = %q, want nil", tmpDir, filenameInSubdir, adata)
	}
}

func TestBeneath(t *testing.T) {
	filename := "something.txt"
	edata := []byte("content")

	tmpDir := t.TempDir()

	if err := os.Mkdir(path.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	filenameInSubdir := path.Join("subdir", filename)
	if err := WriteFileBeneath(tmpDir, filenameInSubdir, edata, 0644); err != nil {
		t.Fatal(err)
	}
	adata, err := ReadFileBeneath(tmpDir, filenameInSubdir)
	if err != nil {
		t.Fatal(err)
	}
	if string(adata) != string(edata) {
		t.Errorf("ReadFileAt(%q, %q) = %q, want %q", tmpDir, filenameInSubdir, adata, edata)
	}
}
