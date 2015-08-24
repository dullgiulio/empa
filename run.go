package empa

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"sync"
)

// General configuration options for a empa library user
type Config struct {
	// Number of processes to run locally
	NSubprocs int
	// Number of CPUs to use
	NCPUs int
	// Number of ep processes
	NProcs int
	// Number of this process in the process group
	ProcID int
	// Number of lines to buffer while feeding input
	InCount int
	// Number of lines to buffer while reading output
	OutCount int
	// Number of lines to buffer while reading errors
	ErrCount int
	// Bytes that separates lines of input
	InEOL EOL
	// Bytes that separates lines of output
	OutEOL EOL
}

// Runner represents a runnable task on a single instance.  A Runner can
// launch and manage several workers to process one partition of the
// input data.
type Runner struct {
	conf Config
	fin  chan string
	fout chan string
	ferr chan string
	err  chan error
	done chan struct{}
	wg   *sync.WaitGroup
}

// NewRunner creates a Runner applying a Config.
func NewRunner(c Config) *Runner {
	return &Runner{
		fin:  make(chan string, c.InCount),
		fout: make(chan string, c.OutCount),
		ferr: make(chan string, c.ErrCount),
		err:  make(chan error),
		done: make(chan struct{}),
		wg:   &sync.WaitGroup{},
		conf: c,
	}
}

// Start starts a Cmd subprocess.  Requires its number relative to
// the total subprocesses that are going to be run.
func (r *Runner) Start(i int, c *Cmd) error {
	r.initCmd(c)
	return c.run(uint(i % r.conf.NCPUs))
}

// Wait waits until all subprocesses have exited.
func (r *Runner) Wait() {
	// We will get the done signal twice (stdout, stderr) per process
	for i := 0; i < r.conf.NSubprocs*2; i++ {
		<-r.done
	}
	close(r.fout)
	close(r.ferr)
	// Wait for errors and output to be consumed
	r.wg.Wait()
}

// Run starts the Runner management goroutines.
func (r *Runner) Run(werr, w io.Writer) {
	go r.logerr()
	r.wg.Add(2)
	go r.consume(werr, r.ferr)
	go r.consume(w, r.fout)
}

// WriteErr can be called to print an error message.
func (r *Runner) WriteErr(err error) {
	r.err <- err
}

// WriteString queues string s to be processed by one subprocess.  s is
// only going to be processed if it belongs to the current data partition.
func (r *Runner) WriteString(s string) {
	if r.canProc(s) {
		r.fin <- s
	}
}

// Close signals that the data to be processed is over.
func (r *Runner) Close() {
	close(r.fin)
}

// initCmd configures a Cmd before it is run.
func (r *Runner) initCmd(c *Cmd) {
	c.fin = r.fin
	c.fout = r.fout
	c.ferr = r.ferr
	c.err = r.err
	c.done = r.done
	c.inEOL = r.conf.InEOL
	c.outEOL = r.conf.OutEOL
}

// consume writes into w the strings received from in.  Strings written
// are separated by the OutEOL as configured in Config.
func (r *Runner) consume(w io.Writer, in <-chan string) {
	defer r.wg.Done()
	bw := bufio.NewWriter(w)
	for s := range in {
		if _, err := bw.WriteString(s); err != nil {
			r.err <- err
			return
		}
		bw.WriteByte(byte(r.conf.OutEOL))
		if err := bw.Flush(); err != nil {
			r.err <- err
			return
		}
	}
}

// logerr prints all errors received from goroutines to stderr.
func (r *Runner) logerr() {
	for err := range r.err {
		if err != nil {
			log.Print(err)
		}
	}
}

// canProc determines if a string s belongs to the current data partition.
func (r *Runner) canProc(s string) bool {
	if r.conf.NProcs < 2 {
		return true
	}
	return sumstring(s)%r.conf.NProcs == r.conf.ProcID
}

// sumstrings sums all bytes of a string.
func sumstring(s string) int {
	var t int
	for _, r := range s {
		t += int(r)
	}
	return t
}

// EOL represents the byte that ends a line.
type EOL byte

// dropCR drops a terminal \r from the data if data is separated by \n
func (e EOL) dropCR(data []byte) []byte {
	if byte(e) == '\n' && len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

// ScanLines implements the bufio.SplitFunc needed by a bufio.Scanner for an EOL object.
func (e EOL) ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, byte(e)); i >= 0 {
		// We have a full eol-terminated line.
		return i + 1, e.dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), e.dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}
