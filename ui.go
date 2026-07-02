package main

import (
	"fmt"
	"sync"
	"time"
)

type pulser struct {
	prefix string
	speed  float64
	mu     sync.RWMutex
	stopCh chan struct{}
	done   sync.WaitGroup
	frames []string
}

func newPulser(prefix string) *pulser {
	p := &pulser{prefix: prefix, stopCh: make(chan struct{})}
	p.frames = barFrames(8)
	p.done.Add(1)
	go func() {
		defer p.done.Done()
		for {
			for _, f := range p.frames {
				select {
				case <-p.stopCh:
					return
				default:
				}
				p.mu.RLock()
				s := p.speed
				p.mu.RUnlock()
				line := fmt.Sprintf("\r%s %s", prefix, f)
				if s > 0 {
					line = fmt.Sprintf("\r%s %s  %s", prefix, f, formatSpeed(s))
				}
				fmt.Print(line)
				time.Sleep(pulseDelay(s))
			}
		}
	}()
	return p
}

func (p *pulser) SetSpeed(s float64) {
	p.mu.Lock()
	p.speed = s
	p.mu.Unlock()
}

func (p *pulser) Close() {
	close(p.stopCh)
	p.done.Wait()
}

func barFrames(width int) []string {
	var frames []string
	right := func(i int) (start, length int) {
		switch {
		case i == 0:
			return 0, 1
		case i == 1:
			return 0, 2
		case i == width-2, i == width-1:
			return i - 1, 2
		default:
			return i - 1, 3
		}
	}
	for i := 0; i < width; i++ {
		start, length := right(i)
		b := make([]byte, width)
		for j := range b {
			b[j] = ' '
		}
		for j := start; j < start+length; j++ {
			b[j] = ':'
		}
		frames = append(frames, "["+string(b)+"]")
	}
	for i := width - 2; i > 0; i-- {
		start, length := right(i)
		b := make([]byte, width)
		for j := range b {
			b[j] = ' '
		}
		for j := start; j < start+length; j++ {
			b[j] = ':'
		}
		frames = append(frames, "["+string(b)+"]")
	}
	return frames
}

func pulseDelay(speed float64) time.Duration {
	if speed <= 0 {
		return 180 * time.Millisecond
	}
	d := 200.0 / (1.0 + speed/8.0)
	if d < 20 {
		d = 20
	}
	return time.Duration(d) * time.Millisecond
}

func formatSpeed(mbps float64) string {
	if *byteMode {
		if mbps >= 8000 {
			return fmt.Sprintf("%6.2f GB/s", mbps/8000)
		}
		if mbps >= 8 {
			return fmt.Sprintf("%6.2f MB/s", mbps/8)
		}
		if mbps > 0 {
			return fmt.Sprintf("%6.2f KB/s", mbps*125)
		}
		return "     —  "
	}
	if mbps >= 1000 {
		return fmt.Sprintf("%6.2f Gbps", mbps/1000)
	}
	if mbps >= 1 {
		return fmt.Sprintf("%6.2f Mbps", mbps)
	}
	if mbps > 0 {
		return fmt.Sprintf("%6.2f Kbps", mbps*1000)
	}
	return "     —  "
}
