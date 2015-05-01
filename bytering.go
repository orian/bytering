// Copyright 2015 to Paweł Szczur.  All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package bytering implements a cyclic memory buffer ByteRing.
// It keeps a given number of bytes in a slice. It uses the same slice all the
// time.
//
// The ByteRing structure is thread safe.
//
// Example code:
// 	buf := NewByteRing(10)
// 	buf.Write([]byte("Tutaj"))
// 	buf.Write([]byte("jest"))
// 	buf.Write([]byte("tekst."))
// 	d = make([]byte, 10)
// 	buf.WriteTo(d) // d will contain "jesttekst."
package bytering

import (
	"io"
	"sync"
)

type ByteRing struct {
	b    []byte
	end  int
	full bool
	capacity int

	m sync.RWMutex
}

// NewByteRing creates a new ByteRing of a given size.
func NewByteRing(size int) *ByteRing {
	return &ByteRing{
		b:    make([]byte, size),
		end:  0, // points to the last element+1 wraped by size
		full: false,
		capacity: size,
		m: sync.RWMutex{},
	}
}

func (b *ByteRing) available() int {
	if !b.full {
		return b.end
	}
	return b.capacity
}

// Available returns a number of bytes currently held in buffer.
// After Size() bytes has been written it's equal to Size().
func (b *ByteRing) Available() int {
	b.m.RLock()
	defer b.m.RUnlock()
	return b.available()
}

// Size returns a size of buffer.
func (b *ByteRing) Size() int {
	return b.capacity
}

// Write writes a byte slice into buffer.
func (b *ByteRing) Write(d []byte) (int, error) {
	// we can only fit last b.size bytes
	ld := len(d)
	b.m.Lock()
	defer b.m.Unlock()
	if ld >= b.capacity {
		copy(b.b, d[ld-b.capacity:])
		b.end = 0
		b.full = true
		return ld, nil
	}

	firstIdx := b.end
	beforeRewind := b.capacity - firstIdx
	if beforeRewind >= ld { // can fit into first interval
		n := copy(b.b[firstIdx:], d)
		b.end = (b.end + n) % b.capacity
		return n, nil
	}
	n := copy(b.b[firstIdx:], d[:beforeRewind])
	n += copy(b.b, d[beforeRewind:])
	b.full = true // we wrap, means we are full
	b.end = (b.end + n) % b.capacity
	return n, nil
}

// Reset resets the state of ByteRing to empty.
func (b *ByteRing) Reset() {
	b.m.Lock()
	defer b.m.Unlock()
	b.end = 0
	b.full = false
}

func (b *ByteRing) firstInterval() (int, int) {
	if !b.full {
		return 0, b.end
	}
	return b.end, b.capacity
}

func (b *ByteRing) secondInterval() (int, int) {
	if !b.full {
		panic("if not full, no second interval")
	}
	return 0, b.end
}

// WriteTo writes all data into provided writer.
func (b *ByteRing) WriteTo(w io.Writer) (int, error) {
	b.m.RLock()
	defer b.m.RUnlock()
	start, end := b.firstInterval()
	n, err := w.Write(b.b[start:end])
	if err != nil || !b.full {
		return n, err
	}

	n1 := 0
	n1, err = w.Write(b.b[:start])
	n += n1
	return n, err
}

// ReadFrom reads from a provided reader until reaches io.EOF
func (b *ByteRing) ReadFrom(r io.Reader) (int, error) {
	buf := make([]byte, 256)
	var err error
	n := 0
	for err != nil {
		n1 := 0
		n1, err = r.Read(buf)
		b.Write(buf[:n1])
		n += n1
	}
	if err == io.EOF {
		err = nil
	}
	return n, err
}

// Tail copies last len(dest) bytes into dest argument.
func (b *ByteRing) Tail(dest []byte) int {
	// assert offset < size!
	destSize := len(dest)
	b.m.RLock()
	defer b.m.RUnlock()
	if destSize > b.available() {
		destSize = b.available()
	}
	if b.full && b.end != 0 {
		_, end := b.secondInterval()
		if destSize <= end {
			return copy(dest, b.b[:end])
		}
		destStart := destSize - end
		n := copy(dest[destStart:], b.b[:end])
		destSize -= n
		n += copy(dest[:destStart], b.b[b.capacity-destSize:])
		return n
	}

	start, end := b.firstInterval()
	start = end - destSize
	return copy(dest, b.b[start:end])
}

// Copy copies a len(dest) bytes into dest shifted by offset.
// Offset equal to 0 means the beginning of data (oldest data).
func (b *ByteRing) Copy(dest []byte, offset int) int {
	// assert offset < size!
	b.m.RLock()
	defer b.m.RUnlock()
	availableData := b.available() - offset
	if availableData <= 0 {
		return 0
	}
	destSize := len(dest)
	if destSize > availableData {
		destSize = availableData
	}
	s, e := b.firstInterval()
	s += offset
	n := 0
	if s >= e {
		offset =  s - e
		s, e = b.secondInterval()
		s += offset
	} else if b.full && b.end != 0 && destSize > e-s {
		n = copy(dest, b.b[s:e])
		s, e = b.secondInterval()
	}
	return n + copy(dest[n:], b.b[s:e])
}
