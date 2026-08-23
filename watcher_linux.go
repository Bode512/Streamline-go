//go:build legacywatcher && linux

package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

func vigilarDirectorio(ctx context.Context, ruta string, cola *Cola) error {
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)
	if err != nil {
		return fmt.Errorf("inotify_init1: %w", err)
	}
	defer syscall.Close(fd)

	wd, err := syscall.InotifyAddWatch(fd, ruta, syscall.IN_CREATE|syscall.IN_MOVED_TO)
	if err != nil {
		return fmt.Errorf("inotify_add_watch: %w", err)
	}
	defer syscall.InotifyRmWatch(fd, uint32(wd))

	buf := make([]byte, 64*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := syscall.Read(fd, buf)
		if err != nil {
			if err == syscall.EINTR || err == syscall.EAGAIN {
				continue
			}
			return fmt.Errorf("read inotify: %w", err)
		}
		if n < syscall.SizeofInotifyEvent {
			continue
		}

		offset := 0
		for offset+syscall.SizeofInotifyEvent <= n {
			event := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))

			if event.Len > 0 {
				raw := buf[offset+syscall.SizeofInotifyEvent : offset+syscall.SizeofInotifyEvent+int(event.Len)]
				if i := bytes.IndexByte(raw, 0); i >= 0 {
					raw = raw[:i]
				}
				name := string(raw)

				if event.Mask&(syscall.IN_CREATE|syscall.IN_MOVED_TO) != 0 && EsVideo(name) {
					cola.Enqueue(filepath.Join(ruta, name))
				}
			}

			offset += syscall.SizeofInotifyEvent + int(event.Len)
		}
	}
}
