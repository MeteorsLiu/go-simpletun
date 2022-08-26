package tun

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	defaultTunPath = "/dev/net/tun"
	defaultMTU     = 1500
	ifReqSize      = unix.IFNAMSIZ
)

type Tun struct {
	device     *os.File
	localAddr  string
	remoteAddr string
}

func (t *Tun) Read(b []byte) (n int, err error) {
	return t.device.Read(b)
}

func (t *Tun) Write(b []byte) (int, error) {
	return t.device.Write(b)
}

func (t *Tun) Close() error {
	return t.device.Close()
}

func (t *Tun) LocalAddr() net.Addr {
	return &net.IPAddr{
		IP: net.ParseIP(t.localAddr),
	}
}

func (t *Tun) RemoteAddr() net.Addr {
	return &net.IPAddr{
		IP: net.ParseIP(t.remoteAddr),
	}
}

func (t *Tun) SetDeadline(_ time.Time) error {
	return nil
}

func (t *Tun) SetReadDeadline(_ time.Time) error {
	return nil
}

func (t *Tun) SetWriteDeadline(_ time.Time) error {
	return nil
}

func setMTU(name string, value int) error {

	// open datagram socket
	fd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM|unix.SOCK_CLOEXEC,
		0,
	)
	if err != nil {
		return err
	}

	defer unix.Close(fd)

	// do ioctl call
	var ifr [unix.IFNAMSIZ + 64]byte
	copy(ifr[:], name)
	*(*uint32)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = uint32(n)
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCSIFMTU),
		uintptr(unsafe.Pointer(&ifr[0])),
	)

	if errno != 0 {
		return fmt.Errorf("failed to set MTU of TUN device: %w", errno)
	}

	return nil

}

func setTun(name string, localAddr string) error {
	if err := exec.Command("ip", "addr", "add", localAddr, "dev", name).Run(); err != nil {
		return err
	}
	if err := exec.Command("ip", "link", "set", name, "up").Run(); err != nil {
		return err
	}
	return nil
}

func New(name string, localAddr string, mtu ...int) (net.Conn, error) {
	MTU := defaultMTU
	if len(mtu) > 0 {
		MTU = mtu[0]
		if MTU > 9000 {
			return nil, errors.New("invalid MTU value")
		}
	}

	nfd, err := unix.Open("/dev/net/tun", unix.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	var flags uint16 = unix.IFF_TUN | unix.IFF_NO_PI
	if len(name) > unix.IFNAMSIZ {
		unix.Close(nfd)
		return nil, errors.New("interface name too long")
	}
	var ifr [unix.IFNAMSIZ + 64]byte
	copy(ifr[:], []byte(name))
	*(*uint16)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = flags

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(nfd),
		uintptr(unix.TUNSETIFF),
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		unix.Close(nfd)
		return nil, errno
	}
	err = unix.SetNonblock(nfd, true)
	if err != nil {
		unix.Close(nfd)
		return nil, err
	}

	// Note that the above -- open,ioctl,nonblock -- must happen prior to handing it to netpoll as below this line.

	fd := os.NewFile(uintptr(nfd), "/dev/net/tun")
	setMTU(name, MTU)
	setTun(name, localAddr)
	return &Tun{
		device:    fd,
		localAddr: localAddr,
	}, nil
}
