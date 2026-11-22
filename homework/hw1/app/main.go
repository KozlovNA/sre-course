package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var ready = true

func main() {
	version := os.Getenv("VERSION")
	pod, _ := os.Hostname()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello version=%s pod=%s\n", version, pod)
	})

	http.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if !ready {
			w.WriteHeader(503)
			fmt.Fprintln(w, "not ready")
			return
		}
		fmt.Fprintln(w, "ready")
	})

	http.HandleFunc("/unready", func(w http.ResponseWriter, r *http.Request) {
		ready = false
		fmt.Fprintln(w, "ok")
	})

	http.HandleFunc("/ready-on", func(w http.ResponseWriter, r *http.Request) {
		ready = true
		fmt.Fprintln(w, "ok")
	})

	server := &http.Server{Addr: ":8080"}
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
