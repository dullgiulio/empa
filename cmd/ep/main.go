package main

import (
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/dullgiulio/empa"
)

func makeConfig() empa.Config {
	var c empa.Config
	// Number of real available CPUs
	c.NCPUs = runtime.NumCPU()
	// Number of processes to start
	c.NSubprocs = c.NCPUs
	// Buffer input and output to reduce goroutines switches
	c.InCount = 1024
	c.OutCount = 1024
	c.InEOL = empa.EOL('\n')
	c.OutEOL = empa.EOL('\n')
	return c
}

func main() {
	var (
		dir  string
		file string
		zero bool
	)
	conf := makeConfig()
	flag.StringVar(&dir, "dir", "", "Directory `DIR` to scan as input source")
	flag.StringVar(&file, "file", "", "File `FILE` to read as input source")
	flag.IntVar(&conf.ProcID, "w", 0, "Worker number `N` of this process")
	flag.IntVar(&conf.NProcs, "wg", 0, "Total number `N` of workers that are being run")
	flag.IntVar(&conf.NSubprocs, "p", conf.NSubprocs, "Number of processes `N` to run in parallel")
	flag.BoolVar(&zero, "0", false, "Read input and write to subprocesses lines delimited by null character instead of newline")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		// TODO: Call usage
		log.Fatal("Usage: ep COMMAND [ARGS...]")
	}
	if zero {
		conf.InEOL = empa.EOL('\x00')
	}
	r := empa.NewRunner(conf)
	r.Run(os.Stderr, os.Stdout)
	for i := 0; i < conf.NSubprocs; i++ {
		cmd := empa.NewCmd(args[0], args[1:]...)
		if err := r.Start(i, cmd); err != nil {
			log.Fatal("ep: ", err)
		}
	}
	switch true {
	case dir != "":
		go walkDir(r, dir)
	case file != "":
		f, err := os.Open(file)
		if err != nil {
			log.Fatal("ep: ", err)
		}
		go feedReader(r, f, conf.InEOL)
	default:
		go feedReader(r, os.Stdin, conf.InEOL)
	}
	r.Wait()
}
