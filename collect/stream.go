package collect

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vmihailenco/msgpack"
)

type (
	// Stream all tweets available on a given directory (see multi-writer)
	Stream struct {
		// Glob pattern to select files from a directory
		Glob string
	}

	PacketVisitor func(Packet) (bool, error)
)

func (s *Stream) Stream(visitor PacketVisitor) error {
	files, err := filepath.Glob(s.Glob)
	if err != nil {
		fmt.Errorf("unable to get list of files to stream", err)
	}
	for _, f := range files {
		ok, err := s.streamFile(f, visitor)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}
	return nil
}

func (s *Stream) streamFile(name string, v PacketVisitor) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		fmt.Errorf("unable to open file for streamming: cause %w", err)
	}
	defer f.Close()
	dec := msgpack.NewDecoder(f)
	for {
		var p Packet
		err := dec.Decode(&p)
		if err == io.EOF {
			return true, nil
		}
		if err != nil {
			return true, fmt.Errorf("unable to decode packet from file %v: cause %v", f.Name(), err)
		}
		ok, _ := v(p)
		if !ok {
			return false, nil
		}
	}
	return true, nil
}
