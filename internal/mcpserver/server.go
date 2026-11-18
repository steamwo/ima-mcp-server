package mcpserver

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/steamwo/ima-mcp-server/internal/ima"
)

func New(service *ima.Service, version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "ima-mcp-server", Version: version}, nil)
	registerNoteTools(server, service)
	registerKnowledgeTools(server, service)
	return server
}
