package mcpserver

import (
	"net/http"
	"testing"

	"github.com/steamwo/ima-mcp-server/internal/ima"
)

func TestNewRegistersAllToolsWithoutPanic(t *testing.T) {
	client, err := ima.NewClient("https://ima.qq.com", "test-client", "test-key", "skill_version=1.1.10", &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	service := ima.NewService(client, nil)

	// Tool registration performs JSON Schema inference. A malformed struct tag or
	// unsupported schema type must fail CI here instead of crashing the binary at runtime.
	_ = New(service, "test")
}
