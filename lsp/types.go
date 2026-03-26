package lsp

import "encoding/json"

// --- JSON-RPC ---

type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *ResponseError   `json:"error,omitempty"`
}

type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// --- Initialize ---

type InitializeParams struct {
	RootURI string `json:"rootUri"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

type ServerCapabilities struct {
	TextDocumentSync TextDocumentSyncOptions `json:"textDocumentSync"`
}

type TextDocumentSyncOptions struct {
	OpenClose bool         `json:"openClose"`
	Change    int          `json:"change"` // 1 = Full, 2 = Incremental
	Save      *SaveOptions `json:"save,omitempty"`
}

type SaveOptions struct {
	IncludeText bool `json:"includeText"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// --- Text Document ---

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// --- didOpen ---

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// --- didChange ---

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []TextDocumentContentChange     `json:"contentChanges"`
}

type TextDocumentContentChange struct {
	Text string `json:"text"`
}

// --- didSave ---

type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

// --- didClose ---

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// --- Diagnostics ---

type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"` // 1=Error, 2=Warning, 3=Info, 4=Hint
	Source   string `json:"source"`
	Message  string `json:"message"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Position struct {
	Line      int `json:"line"`      // 0-indexed
	Character int `json:"character"` // 0-indexed
}
