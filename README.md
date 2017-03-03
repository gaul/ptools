ptools
======

ptools provides parallel implementations of Unix tools.  ptools speeds up
operations on high-latency network file systems, e.g., NFS,
[s3fs](https://github.com/s3fs-fuse/s3fs-fuse), and to a lesser extent large
local file systems.  ptools do not support all the GNU or POSIX options.

Currently implemented:

* du

TODO:

* find

Benchmarks
----------

Testing a [goofys](https://github.com/kahing/goofys)-mounted S3 file system
with 100 ms round-trip times on a dual-core, hyper-threaded laptop:

```
$ echo 3 | sudo tee /proc/sys/vm/drop_caches
3
$ time du linux-4.10/ > /dev/null
real    9m28.106s
user    0m0.340s
sys     0m1.919s

$ echo 3 | sudo tee /proc/sys/vm/drop_caches
3
$ time go run du/du.go linux-4.10/ > /dev/null
real    5m26.251s
user    0m1.385s
sys     0m2.747s
```

License
-------

Copyright (C) 2017 Andrew Gaul

Licensed under the Apache License, Version 2.0
