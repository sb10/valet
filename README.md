# valet

## Overview

`valet` is a utility for performing small, but important data management tasks
automatically. Once started, `valet` will continue working until interrupted
by SIGINT (^C) or SIGTERM (kill), when it will stop gracefully.

### Tasks

- Creating up-to-date checksum files

  - Directory hierarchy styles supported
    
    - Any
  
  - File patterns supported
  
    - *.fast5$
    - *.fastq$

  - Checksum file patterns supported
  
    - (data file name).md5

`valet` will monitor a directory hierarchy and locate data files within it that
have no accompanying checksum file, or have a checksum file that is stale.
`valet` will then calculate the checksum and create or update the checksum file.

### Operation

`valet` is a command-line program with online help. Once launched it will
continue to run until signalled with SIGINT (^C) or SIGTERM (kill), when it
will stop by cancelling the filesystem monitor and waiting for any running jobs
to exit.

### Architecture

`valet` identifies filesystem paths as potential work targets, applies a test
to each and then performs the work on those passing the test (i.e. applies a
filter). This process is implemented as three components

- A filesystem monitor to identify work targets

- A set of predicate (filter) functions.

- A set of work functions and a driver to run them.

Further details on each of these elements are below.

#### Filesystem monitor

`valet` monitors filesystem events under a root directory to detect changes.
Additionally, it performs a periodic sweep of all files under the root directory
because events are not guaranteed to be a complete description of changes e.g.
files may be added to a directory before a watch is established, another program
 on the system may exhaust the user's maximum permitted number of monitors, or
 `valet` may simply have been started after the target files were created.

#### Predicate functions

These functions are used to test filesystem paths to see if they are work
targets. If the function returns a true value, the path is forwarded to a work
function. The predicates are permitted to do anything that does not have side
effects on the path argument e.g. matching the path to a glob or regular
expression, testing whether the path is a regular file, directory or symlink,
testing the file size etc.

A basic API toolkit is provided to create new predicates.

#### Work functions

A work function is applied to every path that passes the filter. A number of
these are executed in parallel, each on a different path. The maximum number of
parallel jobs can be controlled from the command line, the default being to run
as many jobs as there are CPUs. Work function failures will be logged and
counted, but will not cause `valet` to terminate. However, once `valet`
terminates it will do so with a non-zero exit code if any work function failed.

`valet` prevents more than one instance of a work function (either of the same
function, or another) from operating on a particular file concurrently.

### Bugs

- There is no manpage.
