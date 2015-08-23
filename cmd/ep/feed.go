package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"

	"github.com/dullgiulio/empa"
)

func walkDir(r *empa.Runner, path string) {
	if err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			r.WriteErr(err)
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		r.WriteString(p)
		return nil
	}); err != nil {
		r.WriteErr(err)
	}
	r.Close()
}

func feedReader(r *empa.Runner, f io.Reader, eol empa.EOL) {
	sc := bufio.NewScanner(f)
	sc.Split(eol.ScanLines)
	for sc.Scan() {
		r.WriteString(sc.Text())
	}
	if err := sc.Err(); err != nil {
		r.WriteErr(err)
	}
	r.Close()
}
