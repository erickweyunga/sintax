package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/erickweyunga/sintax/analyzer"
	"github.com/erickweyunga/sintax/parser"
	"github.com/erickweyunga/sintax/preprocessor"
)

type Server struct {
	reader  *bufio.Reader
	writer  io.Writer
	mu      sync.Mutex
	docs    map[string]string
	rootURI string
}

func Start(stdlibDir string) {
	s := &Server{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
		docs:   make(map[string]string),
	}
	s.run(stdlibDir)
}

func (s *Server) run(stdlibDir string) {
	for {
		msg, err := s.readMessage()
		if err != nil {
			return
		}

		var req Request
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			s.handleInitialize(req)
		case "initialized":
		case "shutdown":
			s.sendResponse(req.ID, nil, nil)
		case "exit":
			return
		case "textDocument/didOpen":
			s.handleDidOpen(req, stdlibDir)
		case "textDocument/didChange":
			s.handleDidChange(req, stdlibDir)
		case "textDocument/didSave":
			s.handleDidSave(req, stdlibDir)
		case "textDocument/didClose":
			s.handleDidClose(req)
		}
	}
}

func (s *Server) readMessage() ([]byte, error) {
	var contentLength int
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			fmt.Sscanf(line, "Content-Length: %d", &contentLength)
		}
	}
	if contentLength == 0 {
		return nil, fmt.Errorf("no content length")
	}
	body := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, body)
	return body, err
}

func (s *Server) sendMessage(msg interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, err := json.Marshal(msg)
	if err != nil {
		return
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	s.writer.Write([]byte(header))
	s.writer.Write(body)
}

func (s *Server) sendResponse(id *json.RawMessage, result interface{}, respErr *ResponseError) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	if respErr != nil {
		resp.Error = respErr
	}
	s.sendMessage(resp)
}

func (s *Server) sendNotification(method string, params interface{}) {
	s.sendMessage(Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}

func (s *Server) handleInitialize(req Request) {
	var params InitializeParams
	json.Unmarshal(req.Params, &params)
	s.rootURI = params.RootURI

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: TextDocumentSyncOptions{
				OpenClose: true,
				Change:    1, // Full sync
				Save:      &SaveOptions{IncludeText: true},
			},
		},
		ServerInfo: &ServerInfo{
			Name:    "sintax-lsp",
			Version: "0.1.0",
		},
	}
	s.sendResponse(req.ID, result, nil)
}

func (s *Server) handleDidOpen(req Request, stdlibDir string) {
	var params DidOpenTextDocumentParams
	json.Unmarshal(req.Params, &params)

	uri := params.TextDocument.URI
	text := params.TextDocument.Text
	s.docs[uri] = text
	s.diagnose(uri, text, stdlibDir)
}

func (s *Server) handleDidChange(req Request, stdlibDir string) {
	var params DidChangeTextDocumentParams
	json.Unmarshal(req.Params, &params)

	uri := params.TextDocument.URI
	if len(params.ContentChanges) > 0 {
		text := params.ContentChanges[0].Text
		s.docs[uri] = text
		s.diagnose(uri, text, stdlibDir)
	}
}

func (s *Server) handleDidSave(req Request, stdlibDir string) {
	var params DidSaveTextDocumentParams
	json.Unmarshal(req.Params, &params)

	uri := params.TextDocument.URI
	text := params.Text
	if text == "" {
		text = s.docs[uri]
	}
	s.diagnose(uri, text, stdlibDir)
}

func (s *Server) handleDidClose(req Request) {
	var params DidCloseTextDocumentParams
	json.Unmarshal(req.Params, &params)

	uri := params.TextDocument.URI
	delete(s.docs, uri)
	s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []Diagnostic{},
	})
}

func (s *Server) diagnose(uri, text, stdlibDir string) {
	filename := uriToPath(uri)
	lines := strings.Split(text, "\n")

	result := preprocessor.Process(text)
	p := parser.NewParser()
	program, err := p.ParseString(filename, result.Source)
	if err != nil {
		s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
			URI: uri,
			Diagnostics: []Diagnostic{
				{
					Range: Range{
						Start: Position{Line: 0, Character: 0},
						End:   Position{Line: 0, Character: 0},
					},
					Severity: 1, // Error
					Source:   "sintax",
					Message:  err.Error(),
				},
			},
		})
		return
	}

	errors := analyzer.Analyze(program, result.Imports, filename, lines, result.LineMap, stdlibDir)

	diagnostics := make([]Diagnostic, 0, len(errors))
	for _, e := range errors {
		severity := 2 // Warning
		if e.Level == "error" {
			severity = 1 // Error
		}

		line := e.Line - 1 // LSP is 0-indexed
		if line < 0 {
			line = 0
		}

		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: line, Character: 0},
				End:   Position{Line: line, Character: len(sourceLine(lines, e.Line))},
			},
			Severity: severity,
			Source:   "sintax",
			Message:  e.Message,
		})
	}

	s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

func sourceLine(lines []string, line int) string {
	if line > 0 && line <= len(lines) {
		return lines[line-1]
	}
	return ""
}

func uriToPath(uri string) string {
	path := strings.TrimPrefix(uri, "file://")
	path = strings.ReplaceAll(path, "%20", " ")
	return filepath.Clean(path)
}

func FindStdlibDir() string {
	if home := os.Getenv("SINTAX_HOME"); home != "" {
		dir := filepath.Join(home, "stdlib")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	if userHome, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(userHome, ".sintax", "stdlib")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}
