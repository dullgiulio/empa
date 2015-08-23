package empa

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"sync"
)

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

type Runner struct {
	conf Config
	fin  chan string
	fout chan string
	ferr chan string
	err  chan error
	done chan struct{}
	wg   *sync.WaitGroup
}

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

func (r *Runner) Start(i int, c *Cmd) error {
	r.initCmd(c)
	return c.run(uint(i % r.conf.NCPUs))
}

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

func (r *Runner) Run(werr, w io.Writer) {
	go r.logerr()
	r.wg.Add(2)
	go r.consume(werr, r.ferr)
	go r.consume(w, r.fout)
}

func (r *Runner) WriteErr(err error) {
	r.err <- err
}

func (r *Runner) WriteString(s string) {
	if r.canProc(s) {
		r.fin <- s
	}
}

func (r *Runner) Close() {
	close(r.fin)
}

func (r *Runner) initCmd(c *Cmd) {
	c.fin = r.fin
	c.fout = r.fout
	c.ferr = r.ferr
	c.err = r.err
	c.done = r.done
	c.inEOL = r.conf.InEOL
	c.outEOL = r.conf.OutEOL
}

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

func (r *Runner) logerr() {
	for err := range r.err {
		if err != nil {
			log.Print(err)
		}
	}
}

func (r *Runner) canProc(s string) bool {
	if r.conf.NProcs < 2 {
		return true
	}
	return sumstring(s)%r.conf.NProcs == r.conf.ProcID
}

func sumstring(s string) int {
	var t int
	for _, r := range s {
		t += int(r)
	}
	return t
}

type EOL byte

// dropCR drops a terminal \r from the data.
func (e EOL) dropCR(data []byte) []byte {
	if byte(e) == '\n' && len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

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
