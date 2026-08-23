package main

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"runtime"
)

func moveToTrash(path string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("papelera no disponible en %s", runtime.GOOS)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(path))
	command := "$p=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('" + encoded + "')); [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile($p, 'OnlyErrorDialogs', 'SendToRecycleBin')"
	return exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command).Run()
}
