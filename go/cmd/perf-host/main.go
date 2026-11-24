// Command perf-host runs the real ERIDION service (journal reader, SQLite
// writer, reducers, providers wiring, sockets, static UI) without a native
// window, for the G3K performance measurements. It is test infrastructure,
// never shipped: synthetic journals and an isolated profile only, and no
// provider is configured, so it makes no network requests.
//
//	perf-host -journal DIR -profile DIR
//
// stdout: PERF_HOST_URL=<loopback URL>, PERF_HOST_PID=<pid>.
// stdin commands: "stats" prints PERF_STATS <json>; "reset" clears counters;
// "quit" stops the service.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/Jakoczig/eridion/go/internal/frontend"
	"github.com/Jakoczig/eridion/go/internal/perfstats"
	"github.com/Jakoczig/eridion/go/internal/webapp"
)

func main() {
	journalDir := flag.String("journal", "", "synthetic journal directory")
	profile := flag.String("profile", "", "isolated, empty-or-reused profile directory")
	flag.Parse()
	if *journalDir == "" || *profile == "" {
		fmt.Fprintln(os.Stderr, "perf-host: -journal and -profile are required")
		os.Exit(2)
	}
	assets, err := frontend.FS()
	if err != nil {
		panic(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	// Same feature set as the desktop app's normal mode (cmd/daedalus).
	server, err := webapp.StartLocalWithOptions(ctx, assets, webapp.Options{StorageDir: *profile, JournalDir: *journalDir, EnableJournal: true, EnableNavigation: true, EnableDashboard: true, EnableParity: true, EnableExploration: true})
	if err != nil {
		panic(err)
	}
	fmt.Println("PERF_HOST_URL=" + server.URL())
	fmt.Printf("PERF_HOST_PID=%d\n", os.Getpid())
	go func() {
		in := bufio.NewScanner(os.Stdin)
		for in.Scan() {
			switch in.Text() {
			case "stats":
				raw, _ := json.Marshal(perfstats.Take())
				fmt.Println("PERF_STATS " + string(raw))
			case "reset":
				perfstats.Reset()
				fmt.Println("PERF_RESET")
			case "quit":
				cancel()
				return
			}
		}
		cancel()
	}()
	<-ctx.Done()
	closeCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	_ = server.Close(closeCtx)
}
