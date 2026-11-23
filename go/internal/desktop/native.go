package desktop

import (
	"bytes"
	"context"
	"errors"
	"github.com/Jakoczig/eridion/go/internal/fieldexport"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type nativeHost struct {
	windowMu      sync.Mutex
	profile       string
	saveMu        sync.Mutex
	saveDashboard func()
	trayAvailable bool
	app           *application.App
	launcher      *application.WebviewWindow
	dashboard     atomic.Pointer[application.WebviewWindow]
	controller    *Controller
	base          string
	companion     bool
	started       atomic.Bool
	requested     atomic.Bool
	stopped       atomic.Bool
	quitOnce      sync.Once
}

func newNativeHost(stop func()) (windowHost, error) {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return nil, errors.New("native desktop needs a graphical session and GTK4/WebKitGTK 6.0; use --smoke-test for offline headless verification")
	}
	host := &nativeHost{}
	host.app = application.New(application.Options{
		Name:        "ERIDION Companion Preview",
		Description: "Native local-first companion",
		Windows:     application.WindowsOptions{DisableQuitOnLastWindowClosed: true},
		Linux:       application.LinuxOptions{DisableQuitOnLastWindowClosed: true},
		// Native Wails uses its custom-scheme handler, not another TCP service.
		// All page assets and RPC traffic use the supplied loopback URL.
		Assets:     application.AssetOptions{Handler: http.NotFoundHandler(), DisableLogging: true},
		OnShutdown: func() { host.stopped.Store(true); stop() },
	})
	host.app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		host.started.Store(true)
		// Screen enumeration is ready here. Apply startup sizing once, never on map changes.
		if host.launcher != nil && !host.launcher.IsMaximised() {
			options := host.dashboardOptions()
			host.launcher.SetSize(options.Width, options.Height)
		}
		host.maybeQuit()
	})
	return host, nil
}

func (h *nativeHost) Bind(c *Controller) {
	h.controller = c
	c.mu.Lock()
	h.profile = c.windowProfile
	c.mu.Unlock()
	c.Bind(h.Action)
	c.BindAvatar(func() ([]byte, error) {
		if !h.started.Load() || h.stopped.Load() || !h.launcher.IsFocused() {
			return nil, errors.New("Focus the native launcher before choosing a portrait")
		}
		path, err := h.app.Dialog.OpenFile().SetTitle("Choose commander portrait").AddFilter("PNG or JPEG", "*.png;*.jpg;*.jpeg").PromptForSingleSelection()
		if err != nil || path == "" {
			return nil, errors.New("Portrait selection cancelled")
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, errors.New("Cannot open selected portrait")
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 512<<10 {
			return nil, errors.New("Choose a PNG or JPEG under 512 KiB")
		}
		return io.ReadAll(io.LimitReader(f, (512<<10)+1))
	})
	c.BindClipboard(func(name string) bool {
		if !h.started.Load() || h.stopped.Load() {
			return false
		}
		return application.InvokeSyncWithResult(func() bool { return h.app.Clipboard.SetText(name) })
	})
	c.BindCSV(func(name, data string) error {
		path, err := h.app.Dialog.SaveFile().SetFilename(name).AddFilter("CSV", "*.csv").PromptForSingleSelection()
		if err != nil {
			return err
		}
		return fieldexport.Write(path, data)
	})
}
func (h *nativeHost) Open(localURL string) error {
	h.trayAvailable = runtime.GOOS != "linux"
	if runtime.GOOS == "linux" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		out, _ := exec.CommandContext(ctx, "gdbus", "call", "--session", "--dest", "org.freedesktop.DBus", "--object-path", "/org/freedesktop/DBus", "--method", "org.freedesktop.DBus.NameHasOwner", "org.kde.StatusNotifierWatcher").Output()
		h.trayAvailable = strings.Contains(string(out), "true")
	}
	h.companion = strings.HasSuffix(localURL, "/launcher")
	h.base = strings.TrimSuffix(localURL, "/launcher")
	initial := h.dashboardOptions()
	width, height := initial.Width, initial.Height
	if h.companion {
		initial.StartState = application.WindowStateNormal
	}
	h.launcher = h.app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "ERIDION — Experimental Go Preview", Width: width, Height: height, StartState: initial.StartState, URL: localURL})
	if !h.companion {
		h.rememberDashboard(h.launcher)
	}
	h.launcher.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if h.companion && h.controller != nil && !h.controller.quitOnClose.Load() && !h.requested.Load() {
			e.Cancel()
			if h.trayAvailable {
				h.launcher.Hide()
			} else {
				h.launcher.Minimise()
			}
		} else {
			h.Quit()
		}
	})
	if h.companion {
		tray := h.app.SystemTray.New()
		icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				if (x-16)*(x-16)+(y-16)*(y-16) < 180 {
					icon.Set(x, y, color.RGBA{255, 184, 0, 255})
				}
			}
		}
		var b bytes.Buffer
		_ = png.Encode(&b, icon)
		tray.SetIcon(b.Bytes())
		tray.SetTooltip("ERIDION companion — journal service running")
		menu := application.NewMenu()
		menu.Add("Show Launcher").OnClick(func(*application.Context) { _ = h.Action("launcher") })
		menu.Add("Open Dashboard").OnClick(func(*application.Context) { _ = h.Action("dashboard") })
		menu.AddSeparator()
		menu.Add("Quit ERIDION").OnClick(func(*application.Context) { h.Quit() })
		tray.SetMenu(menu)
		tray.OnClick(func() { _ = h.Action("launcher") })
	}
	return nil
}
func (h *nativeHost) Action(action string) error {
	switch action {
	case "launcher":
		h.launcher.UnMinimise()
		h.launcher.Show()
		h.launcher.Focus()
	case "hide":
		if h.trayAvailable {
			h.launcher.Hide()
		} else {
			h.launcher.Minimise()
		}
	case "dashboard":
		if !h.companion {
			return errors.New("companion mode required")
		}
		h.windowMu.Lock()
		d := h.dashboard.Load()
		if d == nil {
			options := h.dashboardOptions()
			options.Title = "ERIDION Dashboard — Experimental"
			options.URL = h.base + "/nav/route"
			d = h.app.Window.NewWithOptions(options)
			h.dashboard.Store(d)
			h.rememberDashboard(d)
			d.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
				h.dashboard.CompareAndSwap(d, nil)
			})
		}
		h.windowMu.Unlock()
		d.Show()
		d.Focus()
	case "closeLauncher":
		h.launcher.Close()
	case "closeDashboard":
		d := h.dashboard.Swap(nil)
		if d != nil {
			d.Close()
		}
	case "quit":
		h.Quit()
	default:
		return errors.New("unsupported desktop action")
	}
	return nil
}
func (h *nativeHost) Run() error { defer h.stopped.Store(true); return h.app.Run() }
func (h *nativeHost) Quit() {
	h.saveMu.Lock()
	save := h.saveDashboard
	h.saveMu.Unlock()
	if save != nil {
		save()
	}
	h.requested.Store(true)
	h.maybeQuit()
}
func (h *nativeHost) maybeQuit() {
	if h.started.Load() && h.requested.Load() && !h.stopped.Load() {
		h.quitOnce.Do(func() { h.app.Quit() })
	}
}

func (h *nativeHost) dashboardOptions() application.WebviewWindowOptions {
	workWidth, workHeight := 1600, 1000
	if screen := h.app.Screen.GetPrimary(); screen != nil {
		workWidth, workHeight = screen.WorkArea.Width, screen.WorkArea.Height
	}
	if h.launcher != nil {
		if screen, err := h.launcher.GetScreen(); err == nil && screen != nil {
			workWidth, workHeight = screen.WorkArea.Width, screen.WorkArea.Height
		}
	}
	size := dashboardSize(workWidth, workHeight, loadWindowSize(h.profile))
	state := application.WindowStateNormal
	if size.Maximized {
		state = application.WindowStateMaximised
	}
	return application.WebviewWindowOptions{Width: size.Width, Height: size.Height, StartState: state}
}
func (h *nativeHost) rememberDashboard(w *application.WebviewWindow) {
	var mu sync.Mutex
	options := h.dashboardOptions()
	last := windowSize{Width: options.Width, Height: options.Height, Maximized: options.StartState == application.WindowStateMaximised}
	width, height := w.Size()
	if width > 0 && height > 0 && !w.IsMaximised() && !last.Maximized {
		last.Width, last.Height = width, height
	}
	// Keep normal dimensions in memory; write once on close, never per resize.
	w.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) {
		if w.IsMaximised() || w.IsMinimised() {
			return
		}
		width, height := w.Size()
		mu.Lock()
		last.Width, last.Height = width, height
		mu.Unlock()
	})
	store := func() {
		mu.Lock()
		v := last
		mu.Unlock()
		if err := saveWindowSize(h.profile, v); err != nil {
			h.app.Logger.Error("Unable to save dashboard window size", "error", err)
		}
	}
	h.saveMu.Lock()
	h.saveDashboard = store
	h.saveMu.Unlock()
	w.OnWindowEvent(events.Common.WindowMaximise, func(*application.WindowEvent) { mu.Lock(); last.Maximized = true; mu.Unlock() })
	w.OnWindowEvent(events.Common.WindowUnMaximise, func(*application.WindowEvent) { mu.Lock(); last.Maximized = false; mu.Unlock() })
	w.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
		maximized := w.IsMaximised()
		mu.Lock()
		last.Maximized = maximized
		mu.Unlock()
		store()
	})
}
