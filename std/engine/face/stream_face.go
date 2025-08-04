package face

import (
	"fmt"
	"io"
	"net"
	"os"

	enc "github.com/named-data/ndnd/std/encoding"
	ndn_io "github.com/named-data/ndnd/std/utils/io"
)

// StreamFace is a face that uses a stream connection.
type StreamFace struct {
	baseFace
	network string
	addr    string
	conn    net.Conn
}

// Constructs a new StreamFace configured with the specified network, address, and local setting, defaulting to exit the application with code 106 when the face becomes unavailable.
func NewStreamFace(network string, addr string, local bool) *StreamFace {
	s := &StreamFace{
		baseFace: newBaseFace(local),
		network:  network,
		addr:     addr,
	}

	// Quit app by default when stream face fails
	s.OnDown(func() { os.Exit(106) })

	return s
}

// Returns a string representation of the StreamFace in the format "stream-face (network://address)", combining the underlying network type and address.
func (f *StreamFace) String() string {
	return fmt.Sprintf("stream-face (%s://%s)", f.network, f.addr)
}

// Opens a network connection for the StreamFace, verifies required callbacks are set, and initiates packet reception in a goroutine.
func (f *StreamFace) Open() error {
	if f.IsRunning() {
		return fmt.Errorf("face is already running")
	}

	if f.onError == nil || f.onPkt == nil {
		return fmt.Errorf("face callbacks are not set")
	}

	c, err := net.Dial(f.network, f.addr)
	if err != nil {
		return err
	}

	f.conn = c
	f.setStateUp()
	go f.receive()

	return nil
}

// Closes the StreamFace by transitioning its state to closed and closing the underlying connection if it exists, returning any error encountered during the connection close.
func (f *StreamFace) Close() error {
	if f.setStateClosed() {
		if f.conn != nil {
			return f.conn.Close()
		}
	}

	return nil
}

// Sends a wire-encoded packet through the StreamFace if it is active, using a mutex to ensure thread-safe transmission and returning an error if the send fails or the face is not running.
func (f *StreamFace) Send(pkt enc.Wire) error {
	if !f.IsRunning() {
		return fmt.Errorf("face is not running")
	}

	f.sendMut.Lock()
	defer f.sendMut.Unlock()

	_, err := f.conn.Write(pkt.Join())
	if err != nil {
		return err
	}

	return nil
}

// Receives and processes incoming TLV-encoded packets from the stream connection until the face stops running, handling errors and ensuring the face state is set to down upon completion.
func (f *StreamFace) receive() {
	defer f.setStateDown()

	err := ndn_io.ReadTlvStream(f.conn, func(b []byte) bool {
		f.onPkt(b)
		return f.IsRunning()
	}, nil)

	if f.IsRunning() {
		if err != nil {
			f.onError(err)
		} else {
			f.onError(io.EOF)
		}
	}
}
