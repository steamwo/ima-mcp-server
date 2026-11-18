package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/steamwo/ima-mcp-server/internal/config"
	"github.com/steamwo/ima-mcp-server/internal/ima"
	"github.com/steamwo/ima-mcp-server/internal/mcpserver"
)

var version = "dev"

func main() {
	transport := flag.String("transport", "stdio", "MCP transport: stdio or http")
	listen := flag.String("listen", "127.0.0.1:8080", "listen address for HTTP transport")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	httpClient := &http.Client{Timeout: cfg.Timeout}
	api, err := ima.NewClient(cfg.BaseURL, cfg.ClientID, cfg.APIKey, cfg.OpenAPIContext, httpClient)
	if err != nil {
		log.Fatal(err)
	}
	service := ima.NewService(api, &ima.COSUploader{})
	server := mcpserver.New(service, version)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch *transport {
	case "stdio":
		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
	case "http":
		if err := runHTTP(ctx, server, *listen); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported transport %q: use stdio or http", *transport)
	}
}

func runHTTP(ctx context.Context, server *mcp.Server, listen string) error {
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok
"))
	})

	httpServer := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	log.Printf("ima-mcp-server HTTP transport listening on %s (endpoint /mcp)", listen)
	err := httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
