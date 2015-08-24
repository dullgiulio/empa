package empa

import (
	"bufio"
	"io"
	"os/exec"
)

// Cmd describes a running subprocess.  It is initialize automatically
// before being run by a Runner.
type Cmd struct {
	cmd    *exec.Cmd
	inEOL  EOL
	outEOL EOL
	fin    <-chan string
	fout   chan<- string
	ferr   chan<- string
	err    chan<- error
	done   chan<- struct{}
}

// NewCmd creates a Cmd for executable "name" with optional CLI arguments.
func NewCmd(name string, arg ...string) *Cmd {
	return &Cmd{
		cmd: exec.Command(name, arg...),
	}
}

// run configures the i/o and starts the subprocess.  Cpu is the relative
// number of this subprocess to pin it to one available logical CPU.
func (c *Cmd) run(cpu uint) error {
	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := c.cmd.Start(); err != nil {
		return err
	}
	// Non fatal error if we cannot pin this subprocess to a CPU
	if err := pinToCPU(c.cmd.Process.Pid, cpu); err != nil {
		c.err <- err
	}
	// Read from input channel
	go c.feed(stdin, c.fin)
	// Read stdout into fout.  Also wait for the program to terminate.
	go func() {
		c.consume(stdout, c.fout)
		if err := c.cmd.Wait(); err != nil {
			c.err <- err
		}
	}()
	// Read stderr into ferr.
	go c.consume(stderr, c.ferr)
	return nil
}

// consume reads r into out.  Closes r when finished reading.
func (c *Cmd) consume(r io.ReadCloser, out chan<- string) {
	sc := bufio.NewScanner(r)
	sc.Split(c.outEOL.ScanLines)
	for sc.Scan() {
		out <- sc.Text()
	}
	if err := sc.Err(); err != nil {
		c.err <- err
	}
	r.Close()
	c.done <- struct{}{}
}

// feed writes into w from in.  Closes w when finished writing.
func (c *Cmd) feed(w io.WriteCloser, in <-chan string) {
	bw := bufio.NewWriter(w)
	for s := range in {
		if _, err := bw.WriteString(s); err != nil {
			c.err <- err
			return
		}
		bw.WriteByte(byte(c.inEOL))
		if err := bw.Flush(); err != nil {
			c.err <- err
			return
		}
	}
	w.Close()
}
