package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/ysh86/ft64/d2xx"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: cmd address sizeInKB")
		return
	}

	i, err := strconv.ParseInt(os.Args[1], 0, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid arg: %v\n", os.Args[1])
	}
	address := uint32(i)
	i, err = strconv.ParseInt(os.Args[2], 0, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid arg: %v\n", os.Args[2])
	}
	size := uint32(i)

	verMajor, verMinor, verPatch := d2xx.Version()
	fmt.Printf("d2xx library version: %d.%d.%d\n", verMajor, verMinor, verPatch)

	rom, err := d2xx.OpenROM()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return
	}
	//fmt.Printf("ROM handle: %v\n", rom)
	defer rom.Close()

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

		header, err := rom.Read512(address)
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
		for addr := address; addr < address+size; addr += 512 {
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
