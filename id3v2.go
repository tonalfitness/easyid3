package easyid3

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

// ReadID3 takes a reader that assumes is the start of an ID3 block and
// reads all the frames and data. It only supports v2 and UTF-8 (and likely
// ISO-8859-1 though not tested).
// https://id3.org/id3v2.4.0-structure
func ReadID3(rdr io.Reader) (map[string]string, error) {
	r := bufio.NewReader(rdr)
	prefix, err := r.Peek(3)
	if err != nil {
		return nil, err
	}
	if string(prefix) != "ID3" {
		return nil, fmt.Errorf("ID3 header not found")
	}

	// Header is 10 bytes per spec
	buf := make([]byte, 10)
	_, err = io.ReadAtLeast(r, buf, 10)
	if err != nil {
		return nil, err
	}

	header, err := newID3(buf)
	if err != nil {
		return nil, err
	}

	// limit to the body size
	rdr = io.LimitReader(r, int64(header.Size))

	/* TODO Maybe parse ExtendedHeader */
	if header.ExtendedHeader() {
		_, err = io.ReadAtLeast(rdr, buf, 4)
		if err != nil {
			return nil, err
		}
		extendedSize := synsafeInt(buf[:4])
		// throw away the extended header
		_, err = io.CopyN(io.Discard, rdr, int64(extendedSize-4))
		if err != nil {
			return nil, err
		}
	}
	props := map[string]string{}
	// Read frame Header
	for {
		_, err = io.ReadAtLeast(rdr, buf, 10)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		frame := newFrameHeader(buf)
		frame.ReadData(r)
		//fmt.Printf("Frame: %v\n", frame)
		props[frame.FrameID] = frame.Decoded()
	}
	// Footer just read off the last 10 bytes
	if header.HasFooter() {
		_, err = io.ReadAtLeast(r, buf, 10)
		if err != nil {
			return nil, err
		}
	}
	return props, nil
}

type frame struct {
	FrameID string
	Size    int
	Flags   []byte // 2
	Data    []byte
}

func (f *frame) String() string {
	return fmt.Sprintf("%s:%s", f.FrameID, f.Decoded())
}

func (f *frame) Decoded() string {
	if f.Data == nil {
		return ""
	}
	switch f.Data[0] {
	case 0:
		//ISO-8859-1 FIXME?
		return string(f.Data[1 : len(f.Data)-1])
	case 1:
		// UTF-16 TODO
	case 2:
		// UTF-16BE TODO
	case 3:
		// UTF-8 remove first and last bytes
		return string(f.Data[1 : len(f.Data)-1])
	}
	return string(f.Data)
}

func (f *frame) ReadData(r io.Reader) error {
	f.Data = make([]byte, f.Size)
	_, err := io.ReadAtLeast(r, f.Data, f.Size)
	return err
}

// NewFrameHeader takes a raw 10 bytes to parse the frame header
// pass the reader directly to ReadData to get the data
func newFrameHeader(raw []byte) *frame {
	return &frame{
		FrameID: string(raw[:4]),
		Size:    synsafeInt(raw[4:8]),
		Flags:   raw[8:],
	}
}

// this is some ridiculous shit about only using 7 bits
func synsafeInt(bs []byte) int {
	var acc int
	for i := 0; i < len(bs); i++ {
		si := byte(i)
		b := bs[len(bs)-i-1]
		acc |= int(b) << (si * 7)
	}
	return acc
}

type iD3Header struct {
	ID3     string
	Version []byte // 2
	Flags   byte
	Size    int
}

// NewID3 takes a raw 10 bytes to parse the header
func newID3(raw []byte) (*iD3Header, error) {
	if string(raw[:3]) != "ID3" && string(raw[:3]) != "3DI" {
		return nil, errors.New("not an ID3 block")
	}
	return &iD3Header{
		ID3:     string(raw[:3]),
		Version: raw[3:5],
		Flags:   raw[5],
		Size:    synsafeInt(raw[6:]),
	}, nil
}

func (ih *iD3Header) VersionString() string {
	return fmt.Sprintf("2.%d.%d", ih.Version[0], ih.Version[1])
}

func (ih *iD3Header) ExtendedHeader() bool {
	return ih.Flags&1<<6 != 0
}

func (ih *iD3Header) Unsynchronisation() bool {
	return ih.Flags&1<<7 != 0
}
func (ih *iD3Header) Experimental() bool {
	return ih.Flags&1<<5 != 0
}
func (ih *iD3Header) HasFooter() bool {
	return ih.Flags&1<<4 != 0
}

func (ih *iD3Header) IsFooter() bool {
	return ih.ID3 == "3DI"
}
