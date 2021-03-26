package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/ysh86/ft64/d2xx"
)

func main() {
	verMajor, verMinor, verPatch := d2xx.Version()
	fmt.Printf("d2xx library version: %d.%d.%d\n", verMajor, verMinor, verPatch)

	rom, err := d2xx.OpenRom()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	//fmt.Printf("ROM handle: %v\n", rom)
	defer rom.CloseROM()

	devType, venID, devID := rom.DevInfo()
	fmt.Printf("DevType: %v(%d), vendor ID: 0x%04x, device ID: 0x%04x\n", devType, devType, venID, devID)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("ready?> ")
		_, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			break
		}

		header, err := rom.Read512(0x1000_0000)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			break
		}

		// dump header
		for j := 0; j < len(header); j += 16 {
			for i := 0; i < 16; i++ {
				fmt.Printf("%02x, ", header[j+i])
			}
			fmt.Printf("\n")
		}

		// dump all
		w, err := os.Create("rom.rom")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			break
		}
		defer w.Close()
		for addr := uint32(0x1000_0000); addr < 0x1000_0000+128*1024; addr += 512 {
			data, err := rom.Read512(addr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s\n", err)
				break
			}
			w.Write(data)
		}
	}
	fmt.Println("done")
}
