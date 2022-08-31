package tun

import (
	"testing"
	"time"
)

func TestTun(ts *testing.T) {
	t, err := New("tun0", "10.0.0.1", "10.0.0.2")
	if err != nil {
		ts.Errorf("%v", err)
	}
	defer t.Close()

	go func() {
		b := make([]byte, 1500)
		for {
			n, err := t.Read(b)
			if err != nil {
				return
			}
			ts.Logf("Len: %d, Read: %d", len(b), n)
		}
	}()
	<-time.After(5 * time.Minute)
}
