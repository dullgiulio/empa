package empa

import (
	"bufio"
	"io"
	"os/exec"
)

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

func NewCmd(name string, arg ...string) *Cmd {
	return &Cmd{
		cmd: exec.Command(name, arg...),
	}
}

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
	go c.feed(stdin, c.fin)
	go func() {
		c.consume(stdout, c.fout)
		if err := c.cmd.Wait(); err != nil {
			c.err <- err
		}
	}()
	go c.consume(stderr, c.ferr)
	return nil
}

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
