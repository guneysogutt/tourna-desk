package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"tourna-desk/internal/database"
	"tourna-desk/internal/handler"
	"tourna-desk/internal/service"
)

//go:embed dist/*
var embeddedDist embed.FS

func main() {
	port := flag.String("port", "8080", "HTTP server port")
	dbPath := flag.String("db", "tourna.db", "SQLite database file path")
	noBrowser := flag.Bool("no-browser", false, "Disable auto-opening browser on startup")
	flag.Parse()

	if envPort := os.Getenv("PORT"); envPort != "" && *port == "8080" {
		*port = envPort
	}

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database (%s): %v", *dbPath, err)
	}
	defer db.Close()

	localIP := detectLocalIP()
	svc := service.NewTournamentService(db)
	apiHandler := handler.NewAPIHandler(svc, localIP, *port)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	// Serve embedded frontend static assets with SPA fallback
	distFS, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		log.Fatalf("Failed to mount embedded dist filesystem: %v", err)
	}
	fileServer := http.FileServer(http.FS(distFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath == "" || cleanPath == "." {
			cleanPath = "index.html"
		}

		if f, err := distFS.Open(cleanPath); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA Fallback to index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	localURL := fmt.Sprintf("http://localhost:%s", *port)
	networkURL := fmt.Sprintf("http://%s:%s", localIP, *port)

	printStartupBanner(*dbPath, localURL, networkURL)

	if !*noBrowser && os.Getenv("NO_BROWSER") == "" {
		go func() {
			time.Sleep(450 * time.Millisecond)
			openBrowser(localURL)
		}()
	}

	addr := ":" + *port
	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept-Language")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func detectLocalIP() string {
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip == nil || ip.IsLoopback() {
					continue
				}
				ip4 := ip.To4()
				if ip4 != nil {
					return ip4.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func printStartupBanner(dbPath, localURL, networkURL string) {
	fmt.Println("================================================================")
	fmt.Println("  TOURNA-DESK | On-Premise Sports Tournament & Fixture Manager  ")
	fmt.Println("================================================================")
	fmt.Printf("  Database   : %s (SQLite WAL Mode)\n", dbPath)
	fmt.Printf("  Local Desk : %s\n", localURL)
	fmt.Printf("  Wi-Fi LAN  : %s  (Mobile Referee / Scorer Access)\n", networkURL)
	fmt.Println("================================================================")
}
