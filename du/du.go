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

// TODO: -h, --human-readable
// TODO: -x, --one-file-system
// TODO: test symlinks

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/abiosoft/semaphore"
	flag "github.com/ogier/pflag"
)

// TODO: -k
var blockSize = flag.IntP("block-size", "B", 1024, "scale sizes by SIZE before printing them")
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
		fmt.Printf("%d\t%s\n", atomic.LoadInt64(&sum), name)
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

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		args = append(args, ".")
	}

	err, sum := Du(args...)
	if *summarize {
		fmt.Printf("%d\t%s\n", sum, args[0])
	}
	if err != nil {
		os.Exit(1)
	}
}
