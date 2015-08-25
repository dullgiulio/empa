# ep: Embarrasingly Parallel

## About

```ep``` is a simple command line utility to distribute parallel work to a set of worker subprocesses.

```ep``` can be used as prefix of another command that reads lines from standard input, for example a list of
files to work on or some data to parse.  All output from all workers is displayed to standard output, along
with any errors on stderr.

```ep``` will automatically start one worked per available CPU and (on Linux) assign a CPU affinity to
pin each worker to one physical CPU.

```ep``` can also be run on several hosts at the same time.  In this case, you need to instruct ```ep``` of
the total number of ```ep``` instances you intend to run and the relative number of each instance.  In this
case, ```ep``` will automatically partition the input between the instances.  Input data must be the same
on all hosts, or some data might never be processed.

## Multi-node Partitioning

When running in multi-node mode (```-w``` and ```-wg``` command line options), the data in input is
ignored or processed depending on the result of a basic hashing of the data itself (currenty, the numeric
sum of all bytes.)

This guarantees that no two nodes process the same data and a negligible overhead.

## Workers

Workers must be designed to do only two basic things:
 1. Read data to work on from standard input and exit when the input is over;
 2. Print errors to stderr and any useful output to stdout (or anywhere the user specifies via flags.)

Workers must not exit immediately by cycle through the input until completion.

Workers don't need to be parallel themselves.  A simple sequential worker will produce optimal results.

Workers should also be able to read null-byte separated lines of input.

## Installation

Currently, you need the Go compiler to be set up.  Binary downloads will follow.  Go 1.5 or newer is
recommended.

```
$ go get github.com/dullgiulio/empa
$ go install github.com/dullgiulio/epma/cli/ep
```

To run the examples below:
```
$ go install github.com/dullgiulio/empa/cli/epsum
```

## Example

 * Single node, multiple processes, using the example utility ```epsum``` to print chechsums of files:
```
$ find /usr/bin -type f | ep epsum -t sum512
```
 * Multiple nodes (three nodes), sharing the same filesystem:
```
host1$ find /data/shared -type f | ep -w 1 -wg 3 epsum -t sum512 >sums1
host2$ find /data/shared -type f | ep -w 2 -wg 3 epsum -t sum512 >sums2
host3$ find /data/shared -type f | ep -w 3 -wg 3 epsum -t sum512 >sums3
$ cat sums? > sums
```

## Bugs / Feature Request

Report on Github: https://github.com/dullgiulio/empa/issues
