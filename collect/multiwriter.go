package collect

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type (
	// DirWriter splits files in sequential chunks
	//
	// Each chunk uses compression to reduce the disk footprint
	DirWriter struct {
		// SplitAtBytes indicates when to split (actual split point might be after it)
		SplitAtBytes int
		// Basedir used to store files
		Basedir string
		// NewBasename returns the name of the next chunk based on the given time
		NewBasename func(time.Time) string

		curWriter *os.File
		curSz     int
		err       error
	}
)

const (
	tmpSuffix = ".txt"
)

func (d *DirWriter) Init() error {
	if len(d.Basedir) == 0 {
		d.Basedir = "."
	}
	d.err = os.MkdirAll(d.Basedir, 0755)
	if d.err != nil {
		return d.err
	}
	d.curWriter, d.err = openNextFile(d.Basedir, basenameFn(d.NewBasename)(time.Now()), 0644)
	return d.err
}

func (d *DirWriter) Write(b []byte) (int, error) {
	if d.err != nil {
		return 0, d.err
	}
	if d.curSz >= d.SplitAtBytes {
		d.err = d.doSplit()
	}
	if d.err != nil {
		return 0, d.err
	}
	_, d.err = d.curWriter.Write(b)
	if d.err != nil {
		return 0, d.err
	}
	d.curSz += len(b)
	return len(b), d.err
}

func (d *DirWriter) doSplit() error {
	err := d.wrapCurrentFile()
	if err != nil {
		return err
	}
	d.curWriter, err = openNextFile(d.Basedir, basenameFn(d.NewBasename)(time.Now()), 0644)
	d.curSz = 0
	return d.err
}

func (d *DirWriter) wrapCurrentFile() error {
	err := d.curWriter.Sync()
	if err != nil {
		return err
	}
	err = d.curWriter.Close()
	if err != nil {
		return err
	}
	return stripTmp(d.curWriter.Name())
}

func basenameFn(fn func(t time.Time) string) func(t time.Time) string {
	if fn == nil {
		return defaultBasenameFn
	}
	return fn
}

func defaultBasenameFn(t time.Time) string {
	return t.Format("20060102_150405")
}

func openNextFile(basedir, basename string, mode os.FileMode) (*os.File, error) {
	if len(basedir) == 0 {
		basedir = "."
	}

	target := filepath.Join(basedir, fmt.Sprintf("%v%v", basename, tmpSuffix))
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return nil, err
	}
	return file, err
}

func stripTmp(fp string) error {
	final := fp[:len(fp)-len(tmpSuffix)]
	return os.Rename(fp, final)
}

func (d *DirWriter) Flush() error {
	if d.err != nil {
		return d.err
	}
	d.err = d.curWriter.Sync()
	return d.err
}

func (d *DirWriter) Close() error {
	d.err = d.Flush()
	if d.err != nil {
		return d.err
	}
	d.err = d.wrapCurrentFile()
	if d.err != nil {
		return d.err
	}
	d.err = os.ErrClosed
	return nil
}
