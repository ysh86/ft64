package d2xx

import (
	"fmt"

	"github.com/ysh86/ft64/d2xx/ftdi"
)

func OpenRom() (*device, error) {
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

	// try MPSSE
	err = dev.tryMpsse()
	if err != nil {
		dev.closeDev()
		return nil, err
	}

	return dev, nil
}

func (d *device) CloseROM() {
	d.closeDev()
}

func (d *device) tryMpsse() error {

	return nil
}

func (d *device) DevInfo() (ftdi.DevType, uint16, uint16) {
	return d.t, d.venID, d.devID
}
