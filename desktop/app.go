package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/skip2/go-qrcode"
)

// App struct
type App struct {
	ctx  context.Context
	core *exec.Cmd
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.StartCore()
}

func (a *App) shutdown(ctx context.Context) {
	a.StopCore()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) ServerURL() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return "http://" + localIP() + ":" + port
}

// QRDataURL genera el QR dentro del proceso Desktop, sin descargarlo por HTTP.
func (a *App) QRDataURL() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	png, err := qrcode.Encode("http://"+localIP()+":"+port+"/", qrcode.Medium, 512)
	if err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

func (a *App) StartCore() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	client := &http.Client{Timeout: 300 * time.Millisecond}
	if response, err := client.Get("http://127.0.0.1:" + port + "/api/health"); err == nil {
		response.Body.Close()
		return nil
	}

	root := findCoreRoot()
	var command *exec.Cmd
	if configured := os.Getenv("STREAMLINE_CORE_PATH"); configured != "" {
		command = exec.Command(configured)
	} else if root != "" {
		command = exec.Command("go", "run", ".")
		command.Dir = root
	} else {
		return fmt.Errorf("no se encontró el Core; configura STREAMLINE_CORE_PATH")
	}
	command.Env = append(os.Environ(), "PORT="+port)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("iniciar Core: %w", err)
	}
	a.core = command
	for attempt := 0; attempt < 30; attempt++ {
		if response, err := client.Get("http://127.0.0.1:" + port + "/api/health"); err == nil {
			response.Body.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("el Core no respondió en el puerto %s", port)
}

func (a *App) StopCore() {
	if a.core != nil && a.core.Process != nil {
		_ = a.core.Process.Kill()
		_, _ = a.core.Process.Wait()
		a.core = nil
	}
}

func findCoreRoot() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for current := workingDir; current != filepath.Dir(current); current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
	}
	return ""
}

func localIP() string {
	if connection, err := net.Dial("udp4", "8.8.8.8:80"); err == nil {
		defer connection.Close()
		if address, ok := connection.LocalAddr().(*net.UDPAddr); ok && address.IP.To4() != nil {
			return address.IP.To4().String()
		}
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	linkLocal := ""
	for _, network := range interfaces {
		if network.Flags&net.FlagUp == 0 || network.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := network.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err == nil && ip.To4() != nil {
				if ip.IsLinkLocalUnicast() {
					linkLocal = ip.String()
					continue
				}
				return ip.String()
			}
		}
	}
	if linkLocal != "" {
		return linkLocal
	}
	return "127.0.0.1"
}
