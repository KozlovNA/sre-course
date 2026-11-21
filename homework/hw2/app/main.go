package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

func newHandler(version, pod string) http.Handler {
	mux := http.NewServeMux()
	var ready atomic.Bool
	ready.Store(true)
	var newGreeting atomic.Bool

	hello := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello version=%s pod=%s\n", version, pod)
	}
	mux.HandleFunc("/", hello)
	mux.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if newGreeting.Load() {
			fmt.Fprintf(w, "hello from the new greeting version=%s pod=%s\n", version, pod)
			return
		}
		hello(w, r)
	})

	mux.HandleFunc("/admin/feature", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		enabled := r.URL.Query()["enabled"]
		if len(enabled) != 1 || (enabled[0] != "true" && enabled[0] != "false") {
			http.Error(w, "use enabled=true or enabled=false", http.StatusBadRequest)
			return
		}
		newGreeting.Store(enabled[0] == "true")
		fmt.Fprintf(w, "new-greeting=%s\n", enabled[0])
	})

	mux.HandleFunc("/feature", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintf(w, "new-greeting=%t\n", newGreeting.Load())
	})

	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			w.WriteHeader(503)
			fmt.Fprintln(w, "not ready")
			return
		}
		fmt.Fprintln(w, "ready")
	})

	mux.HandleFunc("/unready", func(w http.ResponseWriter, r *http.Request) {
		ready.Store(false)
		fmt.Fprintln(w, "ok")
	})

	mux.HandleFunc("/ready-on", func(w http.ResponseWriter, r *http.Request) {
		ready.Store(true)
		fmt.Fprintln(w, "ok")
	})

	return mux
}

func main() {
	version := os.Getenv("VERSION")
	pod, _ := os.Hostname()
	server := &http.Server{Addr: ":8080", Handler: newHandler(version, pod)}
	go server.ListenAndServe()
	fmt.Println("started", version, pod)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	fmt.Println("got SIGTERM")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	fmt.Println("stopped")
}
