package tun

import (
	"testing"
	"time"
)

func TestTun(ts *testing.T) {
	t := New("tun0", "10.0.0.1", "10.0.0.2")
	defer t.Close()

	go func() {
		b := make([]byte, 1500)
		for {
			n, err := t.Read(b)
			if err != nil {
				return
			}
			ts.Log(len(n))
		}
	}()
	<-time.After(5 * time.Minute)
}
