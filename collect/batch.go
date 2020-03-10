package collect

import (
	"io"

	"github.com/dghubble/go-twitter/twitter"
)

type (
	TweetBatch struct {
		Items []*twitter.Tweet
	}

	Packet struct {
		Type string
		Buf  []byte
	}
)

func NewPacket(tp string, value interface{}, encoder func(interface{}) ([]byte, error)) (Packet, error) {
	p := Packet{
		Type: tp,
	}
	var err error
	p.Buf, err = encoder(value)
	if err != nil {
		return Packet{}, err
	}
	return p, nil
}

func WritePacket(w io.Writer, p Packet, encoder func(interface{}) ([]byte, error)) (int, error) {
	buf, err := encoder(p)
	if err != nil {
		return 0, err
	}
	return w.Write(buf)
}
