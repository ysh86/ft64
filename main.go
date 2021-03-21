package main

import (
	"fmt"

	"github.com/ysh86/ft64/d2xx"
)

func main() {
	verMajor, verMinor, verPatch := d2xx.Version()
	fmt.Printf("d2xx library version: %d.%d.%d\n", verMajor, verMinor, verPatch)

	rom, err := d2xx.OpenRom()
	if err != nil {
		panic(err)
	}
	//fmt.Printf("ROM handle: %v\n", rom)
	defer rom.CloseROM()

	devType, venID, devID := rom.DevInfo()
	fmt.Printf("DevType: %v, vendor ID: 0x%04x, device ID: 0x%04x\n", devType, venID, devID)

}
