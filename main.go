package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var (
	dlURL      = flag.String("dl", "https://dl.google.com/linux/direct/google-chrome-stable_current_x86_64.rpm", "download test URL")
	pingURL    = flag.String("ping", "https://www.google.com", "latency test URL")
	uploadURL  = flag.String("ul", "https://www.google.com", "upload test URL")
	duration   = flag.Int("duration", 5, "test duration in seconds")
	concurrent = flag.Int("concurrent", 4, "parallel connections")
	average    = flag.Int("a", 1, "number of test runs to average")
	noUpload   = flag.Bool("no-upload", false, "skip upload test")
	noDownload = flag.Bool("no-download", false, "skip download test")
	byteMode   = flag.Bool("B", false, "display speeds in MB/s instead of Mbps")
	update     = flag.Bool("update", false, "rebuild and reinstall the binary")
)

func findGoMod() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found (run from the project directory)")
}

func main() {
	flag.Parse()

	if *update {
		src, err := findGoMod()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		tmp := filepath.Join(src, ".speedtest-tmp")
		cmd := exec.Command("go", "build", "-o", tmp, ".")
		cmd.Dir = src
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		exe, _ := os.Executable()
		if err := os.Rename(tmp, exe); err != nil {
			cmd = exec.Command("sudo", "mv", tmp, exe)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				os.Remove(tmp)
				os.Exit(1)
			}
		}
		fmt.Println("Updated")
		os.Exit(0)
	}

	var totalPing, totalDl, totalUl float64
	pingN, dlN, ulN := 0, 0, 0

	for i := 0; i < *average; i++ {
		show := *average > 1

		pre := "  ping"
		if show {
			pre = fmt.Sprintf("  ping (%d/%d)", i+1, *average)
		}
		p := newPulser(pre)
		ping, err := MeasureLatency(*pingURL, 3)
		p.Close()
		fmt.Print("\033[2K\r")
		if err == nil {
			totalPing += float64(ping.Microseconds()) / 1000
			pingN++
		}

		if !*noDownload {
			pre := "  ↓"
			if show {
				pre = fmt.Sprintf("  ↓ (%d/%d)", i+1, *average)
			}
			p := newPulser(pre)
			speed, _ := MeasureDownload(*dlURL, time.Duration(*duration)*time.Second, *concurrent, func(s float64) {
				p.SetSpeed(s)
			})
			p.Close()
			fmt.Print("\033[2K\r")
			totalDl += speed
			dlN++
		}

		if !*noUpload {
			pre := "  ↑"
			if show {
				pre = fmt.Sprintf("  ↑ (%d/%d)", i+1, *average)
			}
			p := newPulser(pre)
			speed, _ := MeasureUpload(*uploadURL, time.Duration(*duration)*time.Second, *concurrent, func(s float64) {
				p.SetSpeed(s)
			})
			p.Close()
			fmt.Print("\033[2K\r")
			totalUl += speed
			ulN++
		}
	}

	ap := totalPing / float64(pingN)
	ad := totalDl / float64(dlN)
	au := totalUl / float64(ulN)

	if *average > 1 {
		fmt.Printf("  avg ping %.0f ms  ↓ %s  ↑ %s  (%d runs)\n", ap, formatSpeed(ad), formatSpeed(au), *average)
	} else {
		fmt.Printf("  ping %.0f ms  ↓ %s  ↑ %s\n", ap, formatSpeed(ad), formatSpeed(au))
	}
}


