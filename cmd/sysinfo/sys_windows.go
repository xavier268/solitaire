package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func ramInfo() string {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
	m := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if r, _, _ := proc.Call(uintptr(unsafe.Pointer(&m))); r == 0 {
		return "inconnue"
	}
	return fmt.Sprintf("%.1f Go", float64(m.TotalPhys)/(1<<30))
}

func osInfo() string {
	// "Microsoft Windows [Version 10.0.xxxxx]"
	return run("cmd", "/c", "ver")
}
