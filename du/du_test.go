// Copyright 2017 Andrew Gaul
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

package main

import (
	. "gopkg.in/check.v1"

	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type DuTest struct {
	path string
}

var _ = Suite(&DuTest{})

func (s *DuTest) SetUpTest(c *C) {
	var err error
	s.path, err = ioutil.TempDir("", "du-")
	if err != nil {
		c.Fatal(err)
	}
}

func (s *DuTest) TearDownTest(c *C) {
	os.RemoveAll(s.path)
}

func (s *DuTest) TestEmptyDirectory(c *C) {
	err, sum := Du(s.path)
	c.Assert(err, IsNil)

	size, err := runSystemDu(s.path)
	c.Assert(err, IsNil)
	c.Assert(sum, Equals, size)
}

func (s *DuTest) TestSingleFile(c *C) {
	data := make([]byte, 1)
	filename := filepath.Join(s.path, "file")
	err := ioutil.WriteFile(filename, data, 0644)
	c.Assert(err, IsNil)

	err, sum := Du(filename)
	c.Assert(err, IsNil)

	size, err := runSystemDu(filename)
	c.Assert(err, IsNil)
	c.Assert(sum, Equals, size)
}

func (s *DuTest) TestSingleBigFile(c *C) {
	data := make([]byte, 1024*1024)
	err := ioutil.WriteFile(filepath.Join(s.path, "file"), data, 0644)
	c.Assert(err, IsNil)

	err, sum := Du(s.path)
	c.Assert(err, IsNil)

	size, err := runSystemDu(s.path)
	c.Assert(err, IsNil)
	c.Assert(sum, Equals, size)
}

func (s *DuTest) TestDirectory(c *C) {
	err := os.Mkdir(filepath.Join(s.path, "dir"), 0755)
	c.Assert(err, IsNil)

	err, sum := Du(s.path)
	c.Assert(err, IsNil)

	size, err := runSystemDu(s.path)
	c.Assert(err, IsNil)
	c.Assert(sum, Equals, size)
}

func runSystemDu(path string) (int64, error) {
	out, err := exec.Command("/usr/bin/du", "-k", "-s", path).Output()
	if err != nil {
		return -1, err
	}
	sizeStr := strings.SplitN(string(out), "\t", 2)[0]
	return strconv.ParseInt(sizeStr, 10, 64)
}
