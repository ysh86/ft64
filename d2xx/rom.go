package d2xx

import (
	"fmt"
	"time"

	"github.com/ysh86/ft64/d2xx/ftdi"
)

type rom struct {
	dev      *device
	commands [4096]byte
}

func OpenRom() (*rom, error) {
	// open 1st dev
	num, err := numDevices()
	if err != nil {
		return nil, err
	}
	if num < 1 {
		return nil, fmt.Errorf("numDevices: %d", num)
	}
	dev, err := openDev(d2xxOpen, 0)
	if err != nil {
		return nil, err
	}

	// configure device for MPSSE
	err = dev.reset()
	if err != nil {
		dev.closeDev()
		return nil, err
	}
	err = dev.setupCommon()
	if err != nil {
		dev.closeDev()
		return nil, err
	}
	err = dev.setBitMode(0, bitModeMpsse)
	if err != nil {
		dev.closeDev()
		return nil, err
	}

	r := &rom{dev: dev}
	time.Sleep(50 * time.Millisecond)

	// try MPSSE
	err = r.tryMpsse()
	if err != nil {
		r.CloseROM()
		return nil, err
	}

	return r, nil
}

func (r *rom) CloseROM() {
	r.dev.setBitMode(0, bitModeReset)
	r.dev.closeDev()
}

func (r *rom) tryMpsse() error {
	b := 0
	e := 0

	// Enable loopback
	r.commands[e] = 0x84
	e++
	sent, err := r.dev.write(r.commands[b:e])
	if err != nil {
		return err
	}
	if sent != e-b {
		return fmt.Errorf("failed to write command: 0x%02x", r.commands[b])
	}
	b++
	// Check the receive buffer is empty
	n, err := r.dev.read(r.commands[e : e+1])
	if n != 0 || err != nil {
		return fmt.Errorf("MPSSE receive buffer should be empty")
	}

	// Synchronize the MPSSE
	r.commands[e] = 0xab // bogus command
	e++
	sent, err = r.dev.write(r.commands[b:e])
	b++
	for n == 0 && err == nil {
		n, err = r.dev.read(r.commands[e : e+2])
	}
	if err != nil {
		return err
	}
	if n != 2 || r.commands[e] != 0xfa || r.commands[e+1] != 0xab {
		return fmt.Errorf("failed to synchronize the MPSSE")
	}

	// Disable loopback
	r.commands[e] = 0x85
	e++
	sent, err = r.dev.write(r.commands[b:e])
	if err != nil {
		return err
	}
	if sent != e-b {
		return fmt.Errorf("failed to write command: 0x%02x", r.commands[b])
	}
	b++
	// Check the receive buffer is empty
	n, err = r.dev.read(r.commands[e : e+1])
	if n != 0 || err != nil {
		return fmt.Errorf("MPSSE receive buffer should be empty")
	}

	return nil
}

func (r *rom) DevInfo() (ftdi.DevType, uint16, uint16) {
	return r.dev.t, r.dev.venID, r.dev.devID
}
