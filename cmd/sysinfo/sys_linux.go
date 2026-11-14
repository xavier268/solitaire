package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ramInfo() string {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "inconnue"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "MemTotal:" {
			if kb, err := strconv.ParseFloat(f[1], 64); err == nil {
				return fmt.Sprintf("%.1f Go", kb/(1<<20))
			}
		}
	}
	return "inconnue"
}

func osInfo() string {
	return run("uname", "-sr")
}
