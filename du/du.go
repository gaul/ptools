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

// TODO: -x, --one-file-system
// TODO: test symlinks

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/abiosoft/semaphore"
	flag "github.com/ogier/pflag"
)

// TODO: -k
var blockSize = flag.IntP("block-size", "B", 1024, "scale sizes by SIZE before printing them")
var humanReadable = flag.BoolP("human-readable", "h", false, "print sizes in human readable format (e.g., 1K 234M 2G)")
var summarize = flag.BoolP("summarize", "s", false, "display only a total for each argument")

var sem *semaphore.Semaphore

func init() {
	// TODO: size to number of fds
	// TODO: context instead of global
	sem = semaphore.New(512)
}

func Du(name ...string) (error, int64) {
	var wg sync.WaitGroup
	var sum int64
	var err error
	for _, arg := range name {
		wg.Add(1)
		go du(arg, &wg, &sum, &err)
	}
	wg.Wait()
	return err, sum
}

func du(name string, wgWg *sync.WaitGroup, sumSum *int64, err2 *error) error {
	defer wgWg.Done()

	var sum int64
	var wg sync.WaitGroup

	// avoid double release of semaphore
	semReleased := false
	semRelease := func() {
		if !semReleased {
			sem.Release()
			semReleased = true
		}
	}
	sem.Acquire()
	defer semRelease()

	dir, err := os.Open(name)
	if err != nil {
		// TODO: formatting different:
		// du: cannot access ‘/path’: Permission denied
		// du: lstat /path: permission denied
		fmt.Fprintf(os.Stderr, "du: %+v\n", err)
		sem.Release()
		// TODO: atomic set?
		*err2 = err
		return err
	}
	defer dir.Close()

	info, err := dir.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "du: %+v\n", err)
		*err2 = err
		return err
	}

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		atomic.AddInt64(&sum, scaleBlocks(stat.Blocks))
	}

	if info.IsDir() {
		infos, err := dir.Readdir(0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "du: %+v\n", err)
			*err2 = err
			return err
		}
		dir.Close()
		semRelease()

		for _, info := range infos {
			fullname := name + "/" + info.Name()
			if info.IsDir() {
				wg.Add(1)
				go du(fullname, &wg, &sum, err2)
			} else if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				atomic.AddInt64(&sum, scaleBlocks(stat.Blocks))
			}
		}
	}

	wg.Wait()
	atomic.AddInt64(sumSum, atomic.LoadInt64(&sum))
	if !*summarize {
		fmt.Printf("%s\t%s\n", formatBlocks(atomic.LoadInt64(&sum)), name)
	}
	return nil
}

func scaleBlocks(blocks int64) int64 {
	if *blockSize >= 512 {
		return blocks / (int64(*blockSize) / 512)
	} else {
		return blocks * (512 / int64(*blockSize))
	}
}

func formatBlocks(blocks int64) string {
	if *humanReadable {
		bytes := blocks * int64(*blockSize)
		if bytes > (1 << 60) {
			return fmt.Sprintf("%2.1fE", float64(bytes)/(1<<60))
		} else if bytes > 10*(1<<50) {
			return fmt.Sprintf("%dP", bytes/(1<<50))
		} else if bytes > (1 << 50) {
			return fmt.Sprintf("%2.1fP", float64(bytes)/(1<<50))
		} else if bytes > 10*(1<<40) {
			return fmt.Sprintf("%dT", bytes/(1<<40))
		} else if bytes > (1 << 40) {
			return fmt.Sprintf("%2.1fT", float64(bytes)/(1<<40))
		} else if bytes > 10*(1<<30) {
			return fmt.Sprintf("%dG", bytes/(1<<30))
		} else if bytes > (1 << 30) {
			return fmt.Sprintf("%2.1fG", float64(bytes)/(1<<30))
		} else if bytes > 10*(1<<20) {
			return fmt.Sprintf("%dM", bytes/(1<<20))
		} else if bytes > (1 << 20) {
			return fmt.Sprintf("%2.1fM", float64(bytes)/(1<<20))
		} else if bytes > 10*(1<<10) {
			return fmt.Sprintf("%dK", bytes/(1<<10))
		} else if bytes > (1 << 10) {
			return fmt.Sprintf("%2.1fK", float64(bytes)/(1<<10))
		} else {
			return strconv.FormatInt(bytes, 10)
		}
	} else {
		return strconv.FormatInt(blocks, 10)
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		args = append(args, ".")
	}

	err, sum := Du(args...)
	if *summarize {
		fmt.Printf("%s\t%s\n", formatBlocks(sum), args[0])
	}
	if err != nil {
		os.Exit(1)
	}
}
