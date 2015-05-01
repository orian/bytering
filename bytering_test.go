package bytering

import (
	"bytes"
	"testing"
)

func TestInit(t *testing.T) {
	b := NewByteRing(10)
	if b == nil {
		t.Errorf("NewByteRing returns nil")
	}
	want := "alfa"
	b.Write([]byte(want))
	buf := &bytes.Buffer{}
	b.WriteTo(buf)
	if got := buf.String(); want != got {
		t.Errorf("want: %q, got: %q", want, got)
	}
}

func TestExtensive(t *testing.T) {
	var data = []struct {
		Name    string
		BufSize int
		In      []string
		Want    string
	}{
		{"One write smaller than buffer", 10, []string{"Olsztyn"}, "Olsztyn"},
		{"Bigger than buffer", 10, []string{"OlsztynZyje.pl"}, "tynZyje.pl"},
		{"Double write", 10, []string{"Olsztyn", "Zyje.pl"}, "tynZyje.pl"},
		{"big multi write", 10, []string{"Olszt", "ynZyje.pl", " - poz", "ytywna", " stron", "a Olsz", "tyna"}, "a Olsztyna"},
	}

	bbuf := &bytes.Buffer{}
	for i, d := range data {
		buf := NewByteRing(d.BufSize)
		for j, in := range d.In {
			if n, err := buf.Write([]byte(in)); err != nil {
				t.Errorf("[%d] err when writing [%d] text: %s", i, j, err)
				break
			} else if n != len(in) {
				t.Errorf("[%d] could not write full [%d] text, want: %d, got %d", i, j, len(in), n)
				break
			}
		}
		bbuf.Reset()
		buf.WriteTo(bbuf)
		want := d.Want
		if got := bbuf.String(); want != got {
			t.Errorf("[%d] %q with size %d, WriteTo want: %q, got: %q", i, d.Name, d.BufSize, want, got)
		}

		dl := 2
		b := make([]byte, len(want)-dl)
		buf.Tail(b)
		if want[dl:] != string(b) {
			t.Errorf("[%d] %q with size %d, Tail want: %q, got: %q", i, d.Name, d.BufSize, want[dl:], b)
		}

		b = make([]byte, len(want)+2)
		buf.Tail(b)
		if want != string(b[:len(want)]) {
			t.Errorf("[%d] %q with size %d, Tail want: %q, got: %q", i, d.Name, d.BufSize, want, b[:len(want)])
		}

		b = b[:dl]
		for offset := 0; offset <= len(want)-dl; offset += dl {
			w := want[offset : offset+dl]
			if n := buf.Copy(b, offset); n != len(w) {
				t.Errorf("[%d] %q with size %d, Copy, Offset: %d len want: %d, got: %d", i, d.Name, d.BufSize, offset, len(w), n)
			}
			if bytes.Compare([]byte(w), b) != 0 {
				t.Errorf("[%d] %q with size %d, Copy, Offset: %d want: %q, got: %q", i, d.Name, d.BufSize, offset, w, b)
			}
		}
	}
}
