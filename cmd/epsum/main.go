package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/dullgiulio/empa"
)

func init() {
	runtime.GOMAXPROCS(1)
}

func filehash(h hash.Hash, r io.Reader) ([]byte, error) {
	defer h.Reset()
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func proc(h hash.Hash, fname string) error {
	file, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return err
	}
	// Ignore directories
	if fi.IsDir() {
		return nil
	}
	hash, err := filehash(h, file)
	if err != nil {
		return err
	}
	fmt.Printf("%x %s\n", hash, fname)
	return nil
}

func main() {
	var (
		algo string
		h    hash.Hash
		null bool
		eol  empa.EOL = '\n'
	)
	flag.StringVar(&algo, "t", "sha1", "Type `T` of hash function to use; available are: md5, sha1, sha256, sha512")
	flag.BoolVar(&null, "0", false, "Read null separated lines")
	flag.Parse()
	switch strings.ToLower(algo) {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		log.Fatal("Invalid hash type specified; available are: md5, sha1, sha256, sha512")
	}
	if null {
		eol = empa.EOL('\x00')
	}
	sc := bufio.NewScanner(os.Stdin)
	sc.Split(eol.ScanLines)
	for sc.Scan() {
		fname := sc.Text()
		if err := proc(h, fname); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", fname, err)
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal("epsum: ", err)
	}
}
