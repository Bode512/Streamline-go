//go:build legacywatcher && windows

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

type fileNotifyInformation struct {
	NextEntryOffset uint32
	Action          uint32
	FileNameLength  uint32
}

func vigilarDirectorio(ctx context.Context, ruta string, cola *Cola) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createFileW := kernel32.NewProc("CreateFileW")
	createEventW := kernel32.NewProc("CreateEventW")
	readDirectoryChangesW := kernel32.NewProc("ReadDirectoryChangesW")
	getOverlappedResult := kernel32.NewProc("GetOverlappedResult")
	cancelIoEx := kernel32.NewProc("CancelIoEx")

	const (
		fileListDirectory        = 0x0001
		fileShareRead            = 0x00000001
		fileShareWrite           = 0x00000002
		fileShareDelete          = 0x00000004
		openExisting             = 3
		fileFlagBackupSemantics  = 0x02000000
		fileFlagOverlapped       = 0x40000000
		fileNotifyChangeFileName = 0x00000001
		fileNotifyChangeSize     = 0x00000008
		fileActionAdded          = 0x00000001
		fileActionRenamedNewName = 0x00000005
		errorIOPending           = 997
	)

	rutaPtr, err := syscall.UTF16PtrFromString(ruta)
	if err != nil {
		return fmt.Errorf("utf16: %w", err)
	}

	handle, _, lastErr := createFileW.Call(
		uintptr(unsafe.Pointer(rutaPtr)),
		uintptr(fileListDirectory),
		uintptr(fileShareRead|fileShareWrite|fileShareDelete),
		0,
		uintptr(openExisting),
		uintptr(fileFlagBackupSemantics|fileFlagOverlapped),
		0,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return fmt.Errorf("CreateFileW: %v", lastErr)
	}
	h := syscall.Handle(handle)
	defer syscall.CloseHandle(h)

	event, _, eventErr := createEventW.Call(0, 1, 0, 0)
	if event == 0 {
		return fmt.Errorf("CreateEvent: %v", eventErr)
	}
	eventHandle := syscall.Handle(event)
	defer syscall.CloseHandle(eventHandle)

	ov := &syscall.Overlapped{HEvent: eventHandle}
	buf := make([]byte, 64*1024)
	var bytesReturned uint32

	for {
		select {
		case <-ctx.Done():
			cancelIoEx.Call(handle, uintptr(unsafe.Pointer(ov)))
			return ctx.Err()
		default:
		}

		ret, _, opErr := readDirectoryChangesW.Call(
			handle,
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
			1, // bWatchSubtree = TRUE
			uintptr(fileNotifyChangeFileName|fileNotifyChangeSize),
			uintptr(unsafe.Pointer(&bytesReturned)),
			uintptr(unsafe.Pointer(ov)),
			0,
		)
		if ret == 0 && opErr != syscall.Errno(errorIOPending) {
			return fmt.Errorf("ReadDirectoryChangesW: %v", opErr)
		}

		wait, _ := syscall.WaitForSingleObject(eventHandle, 1000)
		if wait == syscall.WAIT_TIMEOUT {
			continue
		}
		if wait != syscall.WAIT_OBJECT_0 {
			return fmt.Errorf("WaitForSingleObject: %d", wait)
		}

		var transferred uint32
		ret, _, opErr = getOverlappedResult.Call(
			handle,
			uintptr(unsafe.Pointer(ov)),
			uintptr(unsafe.Pointer(&transferred)),
			0,
		)
		if ret == 0 {
			return fmt.Errorf("GetOverlappedResult: %v", opErr)
		}
		bytesReturned = transferred

		offset := 0
		for offset < int(bytesReturned) && int(bytesReturned)-offset >= 12 {
			info := (*fileNotifyInformation)(unsafe.Pointer(&buf[offset]))
			if info.FileNameLength > 0 {
				nameLen := int(info.FileNameLength / 2)
				if offset+12+nameLen*2 <= int(bytesReturned) {
					nameSlice := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[offset+12])), nameLen)
					name := syscall.UTF16ToString(nameSlice)

					if info.Action == fileActionAdded || info.Action == fileActionRenamedNewName {
						if EsVideo(name) {
							cola.Enqueue(filepath.Join(ruta, name))
						}
					}
				}
			}

			if info.NextEntryOffset == 0 {
				break
			}
			offset += int(info.NextEntryOffset)
		}
	}
}
