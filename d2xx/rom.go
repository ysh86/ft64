package d2xx

import (
	"fmt"
	"time"

	"github.com/ysh86/ft64/d2xx/ftdi"
)

type rom struct {
	devA     *device
	devB     *device
	commands [8192 * 2]byte
}

func OpenRom() (*rom, error) {
	// open 1st & 2nd dev
	supported := ftdi.FT2232H
	num, err := numDevices()
	if err != nil {
		return nil, err
	}
	if num < 2 {
		return nil, fmt.Errorf("numDevices: %d", num)
	}
	devA, err := openDev(d2xxOpen, 0)
	if err != nil {
		return nil, err
	}
	if devA.t != supported {
		devA.closeDev()
		return nil, fmt.Errorf("device is not %v, but %v", supported, devA.t)
	}
	devB, err := openDev(d2xxOpen, 1)
	if err != nil {
		devA.closeDev()
		return nil, err
	}
	if devB.t != supported {
		devA.closeDev()
		devB.closeDev()
		return nil, fmt.Errorf("device is not %v, but %v", supported, devB.t)
	}

	// configure device for MPSSE
	err = devA.reset()
	if err != nil {
		devA.closeDev()
		devB.closeDev()
		return nil, err
	}
	err = devB.reset()
	if err != nil {
		devA.closeDev()
		devB.closeDev()
		return nil, err
	}
	err = devA.setupCommon()
	if err != nil {
		devA.closeDev()
		devB.closeDev()
		return nil, err
	}
	err = devB.setupCommon()
	if err != nil {
		devA.closeDev()
		devB.closeDev()
		return nil, err
	}
	err = devA.setBitMode(0, bitModeMpsse)
	if err != nil {
		devA.closeDev()
		devB.closeDev()
		return nil, err
	}
	err = devB.setBitMode(0, bitModeMpsse)
	if err != nil {
		devA.closeDev()
		devB.closeDev()
		return nil, err
	}

	r := &rom{devA: devA, devB: devB}
	time.Sleep(50 * time.Millisecond)

	// try MPSSE
	err = r.tryMpsse(r.devA)
	if err != nil {
		r.CloseROM()
		return nil, err
	}
	err = r.tryMpsse(r.devB)
	if err != nil {
		r.CloseROM()
		return nil, err
	}

	// setup GPIO
	err = r.n64SetupPins()
	if err != nil {
		r.CloseROM()
		return nil, err
	}

	// now ready to go
	err = r.n64ResetROM()
	if err != nil {
		r.CloseROM()
		return nil, err
	}

	return r, nil
}

func (r *rom) CloseROM() {
	r.devA.setBitMode(0, bitModeReset)
	r.devB.setBitMode(0, bitModeReset)
	r.devA.closeDev()
	r.devB.closeDev()
	r.devA = nil
	r.devB = nil
}

func (r *rom) DevInfo() (ftdi.DevType, uint16, uint16) {
	return r.devB.t, r.devB.venID, r.devB.devID
}

func (r *rom) Read512(addr uint32) ([]byte, error) {
	err := r.n64SetAddress(addr)
	if err != nil {
		return nil, err
	}

	header, err := r.n64ReadROM512()
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (r *rom) tryMpsse(dev *device) error {
	b := 0
	e := 0

	// Enable loopback
	r.commands[e] = 0x84
	e++
	sent, err := dev.write(r.commands[b:e])
	if err != nil {
		return err
	}
	if sent != e-b {
		return fmt.Errorf("failed to write command: 0x%02x", r.commands[b])
	}
	b++
	// Check the receive buffer is empty
	n, err := dev.read(r.commands[e : e+1])
	if n != 0 || err != nil {
		return fmt.Errorf("MPSSE receive buffer should be empty")
	}

	// Synchronize the MPSSE
	r.commands[e] = 0xab // bogus command
	e++
	_, err = dev.write(r.commands[b:e])
	b++
	for n == 0 && err == nil {
		n, err = dev.read(r.commands[e : e+2])
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
	sent, err = dev.write(r.commands[b:e])
	if err != nil {
		return err
	}
	if sent != e-b {
		return fmt.Errorf("failed to write command: 0x%02x", r.commands[b])
	}
	b++
	// Check the receive buffer is empty
	n, err = dev.read(r.commands[e : e+1])
	if n != 0 || err != nil {
		return fmt.Errorf("MPSSE receive buffer should be empty")
	}

	return nil
}

// n64 pins:
//
//  Channel A:
//   ADBUS0: TCK/SK: OUT (SPI SCLK)
//   ADBUS1: TDI/DO: OUT (SPI MOSI)
//   ADBUS2: TDO/DI: IN  (SPI MISO) // TODO: Not used. It should be output/Lo?
//   ADBUS3: TMS/CS: OUT SPI CS -> Ch.B GPIOL1
//   ADBUS4: GPIOL0: OUT /WE
//   ADBUS5: GPIOL1: OUT /RE
//   ADBUS6: GPIOL2: OUT ALE_L
//   ADBUS7: GPIOL3: OUT ALE_H
//
//   ACBUS0: GPIOH0: I/O AD0
//   ACBUS1: GPIOH1: I/O AD1
//   ACBUS2: GPIOH2: I/O AD2
//   ACBUS3: GPIOH3: I/O AD3
//   ACBUS4: GPIOH4: I/O AD4
//   ACBUS5: GPIOH5: I/O AD5
//   ACBUS6: GPIOH6: I/O AD6
//   ACBUS7: GPIOH7: I/O AD7
//
//  Channel B:
//   BDBUS0: TCK/SK: OUT (SPI SCLK)
//   BDBUS1: TDI/DO: OUT (SPI MOSI)
//   BDBUS2: TDO/DI: IN  (SPI MISO) // TODO: Not used. It should be output/Lo?
//   BDBUS3: TMS/CS: OUT (SPI CS)
//   BDBUS4: GPIOL0: OUT /RST
//   BDBUS5: GPIOL1: IN  WAIT for Ch.A
//   BDBUS6: GPIOL2: OUT CLK
//   BDBUS7: GPIOL3: IN  S_DAT // TODO: Not used. It should be output/Lo? or Pull-up
//
//   BCBUS0: GPIOH0: I/O AD8
//   BCBUS1: GPIOH1: I/O AD9
//   BCBUS2: GPIOH2: I/O AD10
//   BCBUS3: GPIOH3: I/O AD11
//   BCBUS4: GPIOH4: I/O AD12
//   BCBUS5: GPIOH5: I/O AD13
//   BCBUS6: GPIOH6: I/O AD14
//   BCBUS7: GPIOH7: I/O AD15
//
func (r *rom) n64SetupPins() error {
	b := 0
	e := 0

	// clock: master 60_000_000 / ((1+0x0002)*2) [Hz] = 10[MHz]
	// TODO: 7.5[MHz] for flash:3
	clockDivisorHi := uint8(0x00)
	clockDivisorLo := uint8(0x02)
	r.commands[e] = 0x8a // Use 60MHz master clock
	e++
	r.commands[e] = 0x97 // Turn off adaptive clocking
	e++
	r.commands[e] = 0x8c //0x8d // Disable three-phase clocking // TODO
	e++
	r.commands[e] = 0x86 // set clock divisor
	e++
	r.commands[e] = clockDivisorLo
	e++
	r.commands[e] = clockDivisorHi
	e++
	_, err := r.devA.write(r.commands[b:e])
	if err != nil {
		return err
	}
	_, err = r.devB.write(r.commands[b:e])
	if err != nil {
		return err
	}
	b = e

	// pins A
	r.commands[e] = 0x80
	e++
	r.commands[e] = 0b0011_0001 // ALE_H:1, ALE_L:0, /RE:1, /WE:1, CS:0, (MISO:0, MOSI:0, SCLK:1)
	e++
	r.commands[e] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out, (MISO:In, MOSI:Out, SCLK:Out)
	e++
	r.commands[e] = 0x82
	e++
	r.commands[e] = 0x00 // AD7-0:0
	e++
	r.commands[e] = 0xff // AD7-0:Out
	e++
	_, err = r.devA.write(r.commands[b:e])
	if err != nil {
		return err
	}
	b = e

	// pins B
	r.commands[e] = 0x80
	e++
	r.commands[e] = 0b0101_0001 // S_DAT:0, CLK:1, WAIT:0, /RST:1, CS:0, (MISO:0, MOSI:0, SCLK:1)
	e++
	r.commands[e] = 0b0101_1011 // S_DAT:In, CLK:Out, WAIT:In, /RST:Out, CS:Out, (MISO:In, MOSI:Out, SCLK:Out)
	e++
	r.commands[e] = 0x82
	e++
	r.commands[e] = 0x00 // AD15-8:0
	e++
	r.commands[e] = 0xff // AD15-8:Out
	e++
	_, err = r.devB.write(r.commands[b:e])
	if err != nil {
		return err
	}

	return nil
}

func (r *rom) n64ResetROM() error {
	b := 0
	e := 0

	// pins B
	r.commands[e] = 0x80
	e++
	r.commands[e] = 0b0100_0001 // S_DAT, CLK, WAIT, /RST, CS
	e++
	r.commands[e] = 0b0101_1011 // S_DAT:In, CLK:Out, WAIT:In, /RST:Out, CS:Out
	e++
	_, err := r.devB.write(r.commands[b:e])
	if err != nil {
		return err
	}
	b = e
	r.commands[e] = 0x80
	e++
	r.commands[e] = 0b0101_0001 // S_DAT, CLK, WAIT, /RST, CS
	e++
	r.commands[e] = 0b0101_1011 // S_DAT:In, CLK:Out, WAIT:In, /RST:Out, CS:Out
	e++
	_, err = r.devB.write(r.commands[b:e])
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Millisecond)

	return nil
}

func (r *rom) n64SetAddress(addr uint32) error {
	bA := 0
	eA := 0
	bB := 8192
	eB := 8192

	// ALE_H / ALE_L = 0/0 -> wait -> 1/0 -> 1/1,CS:1 -> 1/1,CS:0
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b0011_0001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++
	// wait 0 =  1.6[us]
	// wait 1 =  2.8[us] (+1.2[us]  = 1.20u/byte = 150n/bit)
	// wait 2 =  4.0[us] (+2.4[us]  = 1.20u/byte = 150n/bit)
	// wait 4 =  6.6[us] (+5.0[us]  = 1.25u/byte = 156n/bit)
	// wait 9 = 12.5[us] (+10.9[us] = 1.21u/byte = 151n/bit)
	{
		r.commands[eA] = 0x8f // wait
		eA++
		r.commands[eA] = 9 // uint16 Lo
		eA++
		r.commands[eA] = 0 // uint16 Hi
		eA++
	}
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b1011_0001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++
	// CS:1
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b1111_1001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++
	// CS:0 for delay 200[ns]
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b1111_0001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++

	// Wait On I/O High
	r.commands[eB] = 0x88
	eB++
	// for delay
	r.commands[eB] = 0x80
	eB++
	r.commands[eB] = 0b0101_0001 // S_DAT, CLK, WAIT, /RST, CS
	eB++
	r.commands[eB] = 0b0101_1011 // S_DAT:In, CLK:Out, WAIT:In, /RST:Out, CS:Out
	eB++

	// addr Hi
	r.commands[eB] = 0x82
	eB++
	r.commands[eB] = uint8(addr >> 24) // AD15-8
	eB++
	r.commands[eB] = 0xff // AD15-8:Out
	eB++
	r.commands[eA] = 0x82
	eA++
	r.commands[eA] = uint8((addr >> 16) & 0xff) // AD7-0
	eA++
	r.commands[eA] = 0xff // AD7-0:Out
	eA++
	// ALE_H / ALE_L = 0/1,CS:1
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b0111_1001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++
	// CS:0 for delay
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b0111_0001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++

	// Wait On I/O High
	r.commands[eB] = 0x88
	eB++
	// for delay
	r.commands[eB] = 0x80
	eB++
	r.commands[eB] = 0b0101_0001 // S_DAT, CLK, WAIT, /RST, CS
	eB++
	r.commands[eB] = 0b0101_1011 // S_DAT:In, CLK:Out, WAIT:In, /RST:Out, CS:Out
	eB++

	// addr Lo
	r.commands[eB] = 0x82
	eB++
	r.commands[eB] = uint8((addr >> 8) & 0xff) // AD15-8
	eB++
	r.commands[eB] = 0xff // AD15-8:Out
	eB++
	r.commands[eA] = 0x82
	eA++
	r.commands[eA] = uint8(addr & 0xff) // AD7-0
	eA++
	r.commands[eA] = 0xff // AD7-0:Out
	eA++
	// ALE_H / ALE_L = 0 / 0
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b0011_0001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++

	_, err := r.devB.write(r.commands[bB:eB])
	if err != nil {
		return err
	}
	_, err = r.devA.write(r.commands[bA:eA])
	if err != nil {
		return err
	}

	return nil
}

func (r *rom) n64ReadROM512() ([]byte, error) {
	bA := 0
	eA := 0
	bB := 8192
	eB := 8192

	// Bus direction
	r.commands[eB] = 0x82
	eB++
	r.commands[eB] = 0x00 // AD15-8
	eB++
	r.commands[eB] = 0x00 // AD15-8:In
	eB++
	r.commands[eA] = 0x82
	eA++
	r.commands[eA] = 0x00 // AD7-0
	eA++
	r.commands[eA] = 0x00 // AD7-0:In
	eA++

	for i := 0; i < 256; i++ {
		// /RE = 0
		r.commands[eA] = 0x80
		eA++
		r.commands[eA] = 0b0001_0001 // ALE_H, ALE_L, /RE, /WE, CS
		eA++
		r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
		eA++
		// TODO: flash
		// wait 15 = 1.6u + 150/bit * 8 * 15 = 19.6[us]
		if false {
			r.commands[eA] = 0x8f // wait
			eA++
			r.commands[eA] = 15 // uint16 Lo
			eA++
			r.commands[eA] = 0 // uint16 Hi
			eA++
		}
		// CS:1
		r.commands[eA] = 0x80
		eA++
		r.commands[eA] = 0b0001_1001 // ALE_H, ALE_L, /RE, /WE, CS
		eA++
		r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
		eA++
		// CS:0 for delay
		r.commands[eA] = 0x80
		eA++
		r.commands[eA] = 0b0001_0001 // ALE_H, ALE_L, /RE, /WE, CS
		eA++
		r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
		eA++

		// Wait On I/O High
		r.commands[eB] = 0x88
		eB++
		// for delay
		r.commands[eB] = 0x80
		eB++
		r.commands[eB] = 0b0101_0001 // S_DAT, CLK, WAIT, /RST, CS
		eB++
		r.commands[eB] = 0b0101_1011 // S_DAT:In, CLK:Out, WAIT:In, /RST:Out, CS:Out
		eB++

		// read
		r.commands[eB] = 0x83 // AD15-8
		eB++
		r.commands[eA] = 0x83 // AD7-0
		eA++

		// /RE = 1
		r.commands[eA] = 0x80
		eA++
		r.commands[eA] = 0b0011_0001 // ALE_H, ALE_L, /RE, /WE, CS
		eA++
		r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
		eA++
	}

	// Bus direction
	r.commands[eB] = 0x88 // Wait On I/O High
	eB++
	r.commands[eB] = 0x82
	eB++
	r.commands[eB] = 0x00 // AD15-8
	eB++
	r.commands[eB] = 0xff // AD15-8:Out
	eB++
	// CS:1
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b0011_1001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++
	// CS:0
	r.commands[eA] = 0x80
	eA++
	r.commands[eA] = 0b0011_0001 // ALE_H, ALE_L, /RE, /WE, CS
	eA++
	r.commands[eA] = 0b1111_1011 // ALE_H:Out, ALE_L:Out, /RE:Out, /WE:Out, CS:Out
	eA++
	r.commands[eA] = 0x82
	eA++
	r.commands[eA] = 0x00 // AD7-0
	eA++
	r.commands[eA] = 0xff // AD7-0:Out
	eA++

	_, err := r.devB.write(r.commands[bB:eB])
	if err != nil {
		return nil, err
	}
	bB = eB
	eB += 256
	_, err = r.devA.write(r.commands[bA:eA])
	if err != nil {
		return nil, err
	}
	bA = eA
	eA += 256

	err = r.devB.readAll(r.commands[bB:eB])
	if err != nil {
		return nil, err
	}
	err = r.devA.readAll(r.commands[bA:eA])
	if err != nil {
		return nil, err
	}

	// interleave B(hi) and A(lo)
	result := r.commands[eB : eB+512]
	for i := 0; i < 256; i++ {
		result[i*2+0] = r.commands[bB+i]
		result[i*2+1] = r.commands[bA+i]
	}

	return result, nil
}
