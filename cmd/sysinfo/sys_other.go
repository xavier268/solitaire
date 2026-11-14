//go:build !windows && !linux

package main

func ramInfo() string { return "non disponible sur ce système" }

func osInfo() string { return run("uname", "-sr") }
