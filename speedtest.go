package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type countingWriter struct {
	mu    *sync.Mutex
	total *int64
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.mu.Lock()
	*w.total += int64(n)
	w.mu.Unlock()
	return n, nil
}

func MeasureLatency(url string, count int) (time.Duration, error) {
	var total time.Duration
	client := &http.Client{Timeout: 5 * time.Second}

	for i := 0; i < count; i++ {
		start := time.Now()
		resp, err := client.Head(url)
		if err != nil {
			return 0, err
		}
		resp.Body.Close()
		total += time.Since(start)
	}
	return total / time.Duration(count), nil
}

func MeasureDownload(url string, dur time.Duration, concurrency int, onSpeed func(float64)) (float64, error) {
	client := &http.Client{
		Timeout: dur + 10*time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency,
			MaxConnsPerHost:     concurrency + 2,
			DisableCompression:  true,
		},
	}

	_, err := client.Head(url)
	if err != nil {
		return 0, fmt.Errorf("head request failed: %w", err)
	}

	var totalBytes int64
	var mu sync.Mutex
	stop := make(chan struct{})
	start := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), start.Add(dur))
	defer cancel()

	if onSpeed != nil {
		go func() {
			report := func() {
				mu.Lock()
				b := totalBytes
				mu.Unlock()
				if elapsed := time.Since(start).Seconds(); elapsed > 0.01 {
					onSpeed(float64(b) / elapsed * 8 / 1_000_000)
				}
			}
			report()
			t := time.NewTicker(100 * time.Millisecond)
			defer t.Stop()
			for range t.C {
				select {
				case <-stop:
					return
				default:
				}
				report()
			}
		}()
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 256*1024)
			cw := &countingWriter{mu: &mu, total: &totalBytes}
			for {
				select {
				case <-stop:
					return
				default:
				}
				req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
				resp, err := client.Do(req)
				if err != nil {
					return
				}
				io.CopyBuffer(cw, resp.Body, buf)
				resp.Body.Close()
				if time.Since(start) >= dur {
					return
				}
			}
		}()
	}

	wg.Wait()
	close(stop)

	elapsed := time.Since(start)
	if elapsed > dur {
		elapsed = dur
	}
	if elapsed.Seconds() <= 0 {
		return 0, nil
	}
	bps := float64(totalBytes) * 8 / elapsed.Seconds()
	return bps / 1_000_000, nil
}

func MeasureUpload(url string, dur time.Duration, concurrency int, onSpeed func(float64)) (float64, error) {
	client := &http.Client{
		Timeout: dur + 10*time.Second,
		Transport: &http.Transport{
			MaxIdleConns:    concurrency,
			MaxConnsPerHost: concurrency + 2,
		},
	}

	var totalBytes int64
	var mu sync.Mutex
	stop := make(chan struct{})
	start := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), start.Add(dur))
	defer cancel()

	if onSpeed != nil {
		go func() {
			report := func() {
				mu.Lock()
				b := totalBytes
				mu.Unlock()
				if elapsed := time.Since(start).Seconds(); elapsed > 0.01 {
					onSpeed(float64(b) / elapsed * 8 / 1_000_000)
				}
			}
			report()
			t := time.NewTicker(100 * time.Millisecond)
			defer t.Stop()
			for range t.C {
				select {
				case <-stop:
					return
				default:
				}
				report()
			}
		}()
	}

	data := make([]byte, 64*1024)
	rand.New(rand.NewSource(time.Now().UnixNano())).Read(data)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if time.Since(start) >= dur {
					return
				}
				pr, pw := io.Pipe()
				go func() {
					pw.Write(data)
					pw.Close()
				}()
				req, _ := http.NewRequestWithContext(ctx, "POST", url, pr)
				req.Header.Set("Content-Type", "application/octet-stream")
				resp, err := client.Do(req)
				if err != nil {
					pr.Close()
					return
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				mu.Lock()
				totalBytes += int64(len(data))
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	close(stop)

	elapsed := time.Since(start)
	if elapsed > dur {
		elapsed = dur
	}
	if elapsed.Seconds() <= 0 {
		return 0, nil
	}
	bps := float64(totalBytes) * 8 / elapsed.Seconds()
	return bps / 1_000_000, nil
}
