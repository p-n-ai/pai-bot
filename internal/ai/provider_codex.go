// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

const (
	defaultCodexBaseURL     = "https://chatgpt.com/backend-api"
	defaultCodexAuthBaseURL = "https://auth.openai.com"
	defaultCodexModel       = "gpt-5.4"
	codexOAuthClientID      = "app_EMoamEEZ73f0CkXaXp7hrann"
	maxCodexAuthBytes       = 1 << 20
)

type CodexProvider struct {
	baseURL     string
	authBaseURL string
	client      *http.Client
	authFile    string
	refresher   CodexCredentialRefresher

	mu                sync.Mutex
	accessToken       string
	refreshToken      string
	accountID         string
	explicitAccountID bool
	expiresAt         time.Time
	refreshing        chan struct{}
	refreshErr        error
}

// CodexCredentialRefresher asks Codex app-server to refresh its managed
// ChatGPT login before PaiBot reloads the isolated credential file.
type CodexCredentialRefresher interface {
	Refresh(context.Context) error
}

type codexAuthFile struct {
	AuthMode string      `json:"auth_mode"`
	Tokens   codexTokens `json:"tokens"`
}

type codexTokens struct {
	AccessToken string `json:"access_token"`
	AccountID   string `json:"account_id"`
	IDToken     string `json:"id_token"`
}

type CodexOption func(*CodexProvider)

func WithCodexRefreshToken(token string) CodexOption {
	return func(p *CodexProvider) {
		p.refreshToken = strings.TrimSpace(token)
	}
}

func WithCodexAccountID(accountID string) CodexOption {
	return func(p *CodexProvider) {
		p.accountID = strings.TrimSpace(accountID)
		p.explicitAccountID = p.accountID != ""
	}
}

func WithCodexBaseURL(baseURL string) CodexOption {
	return func(p *CodexProvider) {
		p.baseURL = strings.TrimSpace(baseURL)
	}
}

func WithCodexHTTPClient(client *http.Client) CodexOption {
	return func(p *CodexProvider) {
		p.client = client
	}
}

func WithCodexAuthBaseURL(baseURL string) CodexOption {
	return func(p *CodexProvider) {
		p.authBaseURL = strings.TrimSpace(baseURL)
	}
}

func NewCodexProvider(accessToken string, opts ...CodexOption) (*CodexProvider, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, errors.New("codex access token is required")
	}
	p := &CodexProvider{
		accessToken: accessToken,
		baseURL:     defaultCodexBaseURL,
		authBaseURL: defaultCodexAuthBaseURL,
		client:      http.DefaultClient,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.client == nil {
		return nil, errors.New("codex HTTP client is required")
	}
	if p.baseURL == "" {
		return nil, errors.New("codex base URL is required")
	}
	if p.authBaseURL == "" {
		return nil, errors.New("codex auth base URL is required")
	}

	accountID, expiresAt, err := codexJWTMetadata(accessToken)
	if !p.explicitAccountID {
		if err != nil || accountID == "" {
			return nil, errors.New("failed to extract Codex account ID from access token")
		}
		p.accountID = accountID
	}
	if err == nil {
		p.expiresAt = expiresAt
	}
	return p, nil
}

// NewManagedCodexProvider creates a provider backed by a ChatGPT login owned
// by Codex app-server in an isolated server directory.
func NewManagedCodexProvider(
	authFile string,
	refresher CodexCredentialRefresher,
	opts ...CodexOption,
) (*CodexProvider, error) {
	authFile = strings.TrimSpace(authFile)
	if authFile == "" {
		return nil, errors.New("codex managed auth is not configured")
	}
	if refresher == nil {
		return nil, errors.New("codex managed login refresh is unavailable")
	}
	p := &CodexProvider{
		authFile:    authFile,
		refresher:   refresher,
		baseURL:     defaultCodexBaseURL,
		authBaseURL: defaultCodexAuthBaseURL,
		client:      http.DefaultClient,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.client == nil {
		return nil, errors.New("codex HTTP client is required")
	}
	if p.baseURL == "" {
		return nil, errors.New("codex base URL is required")
	}
	if _, err := p.readManagedCredentials(); err != nil {
		return nil, err
	}
	return p, nil
}

var _ Provider = (*CodexProvider)(nil)
var _ NativeProvider = (*CodexProvider)(nil)

type codexRequest struct {
	Model             string           `json:"model"`
	Store             bool             `json:"store"`
	Stream            bool             `json:"stream"`
	Instructions      string           `json:"instructions"`
	Input             []any            `json:"input"`
	Tools             []codexTool      `json:"tools,omitempty"`
	ToolChoice        string           `json:"tool_choice,omitempty"`
	ParallelToolCalls bool             `json:"parallel_tool_calls,omitempty"`
	MaxOutputTokens   int              `json:"max_output_tokens,omitempty"`
	Temperature       *float64         `json:"temperature,omitempty"`
	Reasoning         *codexReasoning  `json:"reasoning,omitempty"`
	Text              codexTextOptions `json:"text"`
	Include           []string         `json:"include,omitempty"`
	PromptCacheKey    string           `json:"prompt_cache_key,omitempty"`
	headers           map[string]string
}

type codexTextOptions struct {
	Verbosity string               `json:"verbosity"`
	Format    *codexResponseFormat `json:"format,omitempty"`
}

type codexResponseFormat struct {
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict"`
}

type codexReasoning struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type codexTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      *bool           `json:"strict"`
}

type codexResult struct {
	id         string
	model      string
	text       string
	content    []llm.AssistantContent
	toolCalls  []llm.ToolCall
	input      int
	output     int
	cacheRead  int
	cacheWrite int
	status     string
	terminal   bool
}

type codexSSEEvent struct {
	Type        string          `json:"type"`
	OutputIndex int             `json:"output_index"`
	Delta       string          `json:"delta"`
	Arguments   string          `json:"arguments"`
	Item        json.RawMessage `json:"item"`
	Response    json.RawMessage `json:"response"`
	Error       *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type codexResponseItem struct {
	Type             string `json:"type"`
	ID               string `json:"id"`
	CallID           string `json:"call_id"`
	Name             string `json:"name"`
	Arguments        string `json:"arguments"`
	EncryptedContent string `json:"encrypted_content"`
	Summary          []struct {
		Text string `json:"text"`
	} `json:"summary"`
	Content []struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Refusal string `json:"refusal"`
	} `json:"content"`
}

type codexTerminalResponse struct {
	ID     string            `json:"id"`
	Model  string            `json:"model"`
	Status string            `json:"status"`
	Output []json.RawMessage `json:"output"`
	Usage  struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		InputDetails struct {
			CachedTokens     int `json:"cached_tokens"`
			CacheWriteTokens int `json:"cache_write_tokens"`
		} `json:"input_tokens_details"`
	} `json:"usage"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type codexOutputItem struct {
	parsed codexResponseItem
	raw    json.RawMessage
	done   bool
}

func (p *CodexProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	request, err := buildCodexLegacyRequest(req)
	if err != nil {
		return CompletionResponse{}, err
	}
	result, err := p.complete(ctx, request, nil)
	if err != nil {
		return CompletionResponse{}, err
	}
	response := CompletionResponse{
		Content:      result.text,
		Model:        result.responseModel(request.Model),
		InputTokens:  result.input + result.cacheRead + result.cacheWrite,
		OutputTokens: result.output,
	}
	if req.StructuredOutput != nil {
		response.StructuredOutput = json.RawMessage(result.text)
	}
	return response, nil
}

func (p *CodexProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	request, err := buildCodexLegacyRequest(req)
	if err != nil {
		return nil, err
	}
	response, err := p.openResponse(ctx, request)
	if err != nil {
		return nil, err
	}
	chunks := make(chan StreamChunk)
	go func() {
		defer close(chunks)
		defer func() { _ = response.Body.Close() }()
		_, parseErr := parseCodexStream(response.Body, func(delta string) {
			select {
			case chunks <- StreamChunk{Content: delta}:
			case <-ctx.Done():
			}
		})
		if parseErr != nil {
			select {
			case chunks <- StreamChunk{Error: parseErr, Done: true}:
			case <-ctx.Done():
			}
			return
		}
		select {
		case chunks <- StreamChunk{Done: true}:
		case <-ctx.Done():
		}
	}()
	return chunks, nil
}

func (p *CodexProvider) CompleteNative(ctx context.Context, model string, c llm.Context, opts *llm.StreamOptions) (llm.AssistantMessage, error) {
	request, err := buildCodexNativeRequest(model, c, opts)
	if err != nil {
		return llm.AssistantMessage{}, err
	}
	result, err := p.complete(ctx, request, nil)
	if err != nil {
		return llm.AssistantMessage{}, err
	}
	content := result.content
	if len(content) == 0 && result.text != "" {
		content = []llm.AssistantContent{llm.TextContent{Text: result.text}}
	}
	stopReason := llm.StopReasonStop
	if len(result.toolCalls) > 0 {
		stopReason = llm.StopReasonToolUse
	} else if result.status == "incomplete" {
		stopReason = llm.StopReasonLength
	}
	return llm.AssistantMessage{
		Content:       content,
		API:           "openai-codex-responses",
		Provider:      "openai-codex",
		Model:         request.Model,
		ResponseModel: result.responseModel(request.Model),
		ResponseID:    result.id,
		Usage: llm.Usage{
			Input:       result.input,
			Output:      result.output,
			CacheRead:   result.cacheRead,
			CacheWrite:  result.cacheWrite,
			TotalTokens: result.input + result.cacheRead + result.cacheWrite + result.output,
		},
		StopReason: stopReason,
		Timestamp:  time.Now(),
	}, nil
}

func (p *CodexProvider) Models() []ModelInfo {
	return []ModelInfo{{
		ID:          defaultCodexModel,
		Name:        "GPT-5.4 Codex",
		MaxTokens:   128000,
		Description: "OpenAI Codex through a ChatGPT subscription",
	}}
}

func (p *CodexProvider) HealthCheck(ctx context.Context) error {
	token, accountID, err := p.credentials(ctx)
	if err != nil {
		return err
	}
	response, err := p.getModels(ctx, token, accountID)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		_ = response.Body.Close()
		if err := p.refreshCredentials(ctx, token, true); err != nil {
			return fmt.Errorf("codex authentication rejected and token refresh failed: %w", err)
		}
		token, accountID, err = p.credentials(ctx)
		if err != nil {
			return err
		}
		response, err = p.getModels(ctx, token, accountID)
		if err != nil {
			return err
		}
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("codex models API returned status %d", response.StatusCode)
	}
	return nil
}

func (p *CodexProvider) complete(ctx context.Context, request codexRequest, onText func(string)) (codexResult, error) {
	response, err := p.openResponse(ctx, request)
	if err != nil {
		return codexResult{}, err
	}
	defer func() { _ = response.Body.Close() }()
	return parseCodexStream(response.Body, onText)
}

func (p *CodexProvider) openResponse(ctx context.Context, request codexRequest) (*http.Response, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal Codex request: %w", err)
	}
	token, accountID, err := p.credentials(ctx)
	if err != nil {
		return nil, err
	}
	response, err := p.post(ctx, body, request, token, accountID)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusUnauthorized && response.StatusCode != http.StatusForbidden {
		return codexHTTPResponse(response)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	_ = response.Body.Close()

	if err := p.refreshCredentials(ctx, token, true); err != nil {
		return nil, fmt.Errorf("codex authentication rejected and token refresh failed: %w", err)
	}
	token, accountID, err = p.credentials(ctx)
	if err != nil {
		return nil, err
	}
	response, err = p.post(ctx, body, request, token, accountID)
	if err != nil {
		return nil, err
	}
	return codexHTTPResponse(response)
}

func (p *CodexProvider) post(ctx context.Context, body []byte, codexRequest codexRequest, token, accountID string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, codexResponsesURL(p.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create Codex request: %w", err)
	}
	for name, value := range codexRequest.headers {
		request.Header.Set(name, value)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("chatgpt-account-id", accountID)
	request.Header.Set("originator", "pi")
	request.Header.Set("OpenAI-Beta", "responses=experimental")
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Content-Type", "application/json")
	if codexRequest.PromptCacheKey != "" {
		request.Header.Set("session-id", codexRequest.PromptCacheKey)
		request.Header.Set("x-client-request-id", codexRequest.PromptCacheKey)
	}
	response, err := p.client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("send Codex request: %w", err)
	}
	return response, nil
}

func (p *CodexProvider) getModels(ctx context.Context, token, accountID string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, codexModelsURL(p.baseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("create Codex models request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("chatgpt-account-id", accountID)
	request.Header.Set("originator", "pi")
	request.Header.Set("Accept", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("send Codex models request: %w", err)
	}
	return response, nil
}

func codexHTTPResponse(response *http.Response) (*http.Response, error) {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return response, nil
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
	return nil, fmt.Errorf("codex API returned status %d", response.StatusCode)
}

func (p *CodexProvider) credentials(ctx context.Context) (string, string, error) {
	if p.authFile != "" {
		if err := p.refresher.Refresh(ctx); err != nil {
			return "", "", errors.New("refresh Codex managed login")
		}
		credentials, err := p.readManagedCredentials()
		if err != nil {
			return "", "", err
		}
		return credentials.accessToken, credentials.accountID, nil
	}

	p.mu.Lock()
	token := p.accessToken
	expired := !p.expiresAt.IsZero() && !time.Now().Add(30*time.Second).Before(p.expiresAt)
	p.mu.Unlock()
	if expired {
		if err := p.refreshCredentials(ctx, token, false); err != nil {
			return "", "", err
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.accessToken, p.accountID, nil
}

func (p *CodexProvider) refreshCredentials(ctx context.Context, staleToken string, force bool) error {
	if p.authFile != "" {
		if err := p.refresher.Refresh(ctx); err != nil {
			return errors.New("refresh Codex managed login")
		}
		return nil
	}

	for {
		p.mu.Lock()
		if p.accessToken != staleToken {
			p.mu.Unlock()
			return nil
		}
		if !force && (p.expiresAt.IsZero() || time.Now().Add(30*time.Second).Before(p.expiresAt)) {
			p.mu.Unlock()
			return nil
		}
		if p.refreshing != nil {
			done := p.refreshing
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-done:
			}
			p.mu.Lock()
			tokenChanged := p.accessToken != staleToken
			refreshErr := p.refreshErr
			p.mu.Unlock()
			if tokenChanged || refreshErr == nil {
				return nil
			}
			if errors.Is(refreshErr, context.Canceled) || errors.Is(refreshErr, context.DeadlineExceeded) {
				if err := ctx.Err(); err != nil {
					return err
				}
				continue
			}
			return refreshErr
		}
		if p.refreshToken == "" {
			p.mu.Unlock()
			return errors.New("codex refresh token is unavailable")
		}
		done := make(chan struct{})
		p.refreshing = done
		refreshToken := p.refreshToken
		p.mu.Unlock()

		credentials, err := p.requestTokenRefresh(ctx, refreshToken)

		p.mu.Lock()
		if err == nil {
			p.accessToken = credentials.accessToken
			p.refreshToken = credentials.refreshToken
			p.expiresAt = credentials.expiresAt
			if !p.explicitAccountID {
				p.accountID = credentials.accountID
			}
		}
		p.refreshErr = err
		p.refreshing = nil
		close(done)
		p.mu.Unlock()
		return err
	}
}

type codexCredentials struct {
	accessToken  string
	refreshToken string
	accountID    string
	expiresAt    time.Time
}

func (p *CodexProvider) readManagedCredentials() (codexCredentials, error) {
	file, err := os.Open(p.authFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return codexCredentials{}, errors.New("Codex login is not connected; connect it in Admin")
		}
		return codexCredentials{}, errors.New("read Codex managed login")
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxCodexAuthBytes+1))
	if err != nil {
		return codexCredentials{}, errors.New("read Codex managed login")
	}
	if len(data) > maxCodexAuthBytes {
		return codexCredentials{}, errors.New("Codex managed login is too large")
	}
	var auth codexAuthFile
	if err := json.Unmarshal(data, &auth); err != nil {
		return codexCredentials{}, errors.New("Codex managed login is malformed")
	}
	if !strings.EqualFold(strings.TrimSpace(auth.AuthMode), "chatgpt") {
		return codexCredentials{}, errors.New("Codex ChatGPT login is not connected; connect it in Admin")
	}
	accessToken := strings.TrimSpace(auth.Tokens.AccessToken)
	if accessToken == "" {
		return codexCredentials{}, errors.New("Codex ChatGPT login has no access token; reconnect it in Admin")
	}
	accountID := strings.TrimSpace(auth.Tokens.AccountID)
	tokenAccountID, expiresAt, tokenErr := codexJWTMetadata(accessToken)
	if accountID == "" && tokenErr == nil {
		accountID = tokenAccountID
	}
	if accountID == "" && strings.TrimSpace(auth.Tokens.IDToken) != "" {
		if idTokenAccountID, _, err := codexJWTMetadata(strings.TrimSpace(auth.Tokens.IDToken)); err == nil {
			accountID = idTokenAccountID
		}
	}
	if accountID == "" {
		return codexCredentials{}, errors.New("Codex ChatGPT login has no account ID; reconnect it in Admin")
	}
	if tokenErr == nil && !expiresAt.IsZero() && !time.Now().Add(30*time.Second).Before(expiresAt) {
		return codexCredentials{}, errors.New("Codex login is expired; reconnect it in Admin")
	}
	return codexCredentials{
		accessToken: accessToken,
		accountID:   accountID,
		expiresAt:   expiresAt,
	}, nil
}

func (p *CodexProvider) requestTokenRefresh(ctx context.Context, refreshToken string) (codexCredentials, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {codexOAuthClientID},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.authBaseURL, "/")+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return codexCredentials{}, errors.New("create Codex token refresh request")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return codexCredentials{}, ctxErr
		}
		return codexCredentials{}, errors.New("send Codex token refresh request")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return codexCredentials{}, fmt.Errorf("codex token refresh returned status %d", response.StatusCode)
	}
	var decoded struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&decoded); err != nil {
		return codexCredentials{}, errors.New("decode Codex token refresh response")
	}
	if decoded.AccessToken == "" {
		return codexCredentials{}, errors.New("codex token refresh response is missing an access token")
	}
	if decoded.RefreshToken == "" {
		decoded.RefreshToken = refreshToken
	}
	accountID, tokenExpiresAt, err := codexJWTMetadata(decoded.AccessToken)
	if !p.explicitAccountID && (err != nil || accountID == "") {
		return codexCredentials{}, errors.New("failed to extract Codex account ID from refreshed access token")
	}
	expiresAt := tokenExpiresAt
	if decoded.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(decoded.ExpiresIn) * time.Second)
	}
	if expiresAt.IsZero() {
		return codexCredentials{}, errors.New("codex token refresh response is missing an expiry")
	}
	return codexCredentials{
		accessToken:  decoded.AccessToken,
		refreshToken: decoded.RefreshToken,
		accountID:    accountID,
		expiresAt:    expiresAt,
	}, nil
}

func buildCodexLegacyRequest(req CompletionRequest) (codexRequest, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = defaultCodexModel
	}
	request := newCodexRequest(model)
	request.MaxOutputTokens = req.MaxTokens
	if req.Temperature > 0 {
		temperature := req.Temperature
		request.Temperature = &temperature
	}
	var instructions []string
	for i, message := range req.Messages {
		switch message.Role {
		case "system", "developer":
			if len(message.ImageURLs) > 0 {
				return codexRequest{}, fmt.Errorf("codex system message at index %d cannot contain images", i)
			}
			if message.Content != "" {
				instructions = append(instructions, message.Content)
			}
		case "user":
			content := codexUserContent(message.Content, message.ImageURLs)
			if len(content) > 0 {
				request.Input = append(request.Input, map[string]any{"role": "user", "content": content})
			}
		case "assistant":
			if len(message.ImageURLs) > 0 {
				return codexRequest{}, fmt.Errorf("codex assistant message at index %d cannot contain images", i)
			}
			request.Input = append(request.Input, codexAssistantInput(message.Content))
		default:
			return codexRequest{}, fmt.Errorf("unsupported Codex message role at index %d", i)
		}
	}
	request.Instructions = codexInstructions(instructions)
	if err := applyCodexStructuredOutput(&request, req.StructuredOutput); err != nil {
		return codexRequest{}, err
	}
	return request, nil
}

func buildCodexNativeRequest(model string, c llm.Context, opts *llm.StreamOptions) (codexRequest, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultCodexModel
	}
	request := newCodexRequest(model)
	var instructions []string
	if c.SystemPrompt != "" {
		instructions = append(instructions, c.SystemPrompt)
	}
	for _, message := range c.Messages {
		switch typed := message.(type) {
		case llm.SystemMessage:
			if typed.Content != "" {
				instructions = append(instructions, typed.Content)
			}
		case llm.UserMessage:
			content, err := codexNativeUserContent(typed.Content)
			if err != nil {
				return codexRequest{}, err
			}
			if len(content) > 0 {
				request.Input = append(request.Input, map[string]any{"role": "user", "content": content})
			}
		case llm.AssistantMessage:
			for _, block := range typed.Content {
				switch value := block.(type) {
				case llm.TextContent:
					request.Input = append(request.Input, codexAssistantInput(value.Text))
				case llm.ToolCall:
					arguments, err := json.Marshal(value.Arguments)
					if err != nil {
						return codexRequest{}, fmt.Errorf("codex tool call %q arguments: %w", value.Name, err)
					}
					if value.Arguments == nil {
						arguments = []byte("{}")
					}
					callID, itemID, _ := strings.Cut(value.ID, "|")
					item := map[string]any{
						"type":      "function_call",
						"call_id":   callID,
						"name":      value.Name,
						"arguments": string(arguments),
					}
					if itemID != "" {
						item["id"] = itemID
					}
					request.Input = append(request.Input, item)
				case llm.ThinkingContent:
					if value.Signature == "" {
						continue
					}
					item, err := parseCodexReasoningSignature(value.Signature)
					if err != nil {
						return codexRequest{}, err
					}
					request.Input = append(request.Input, item)
				default:
					return codexRequest{}, fmt.Errorf("unsupported Codex assistant content type %T", block)
				}
			}
		case llm.ToolResultMessage:
			output, err := nativeOpenAIToolResultText(typed)
			if err != nil {
				return codexRequest{}, err
			}
			callID, _, _ := strings.Cut(typed.ToolCallID, "|")
			request.Input = append(request.Input, map[string]any{
				"type":    "function_call_output",
				"call_id": callID,
				"output":  output,
			})
		default:
			return codexRequest{}, fmt.Errorf("unsupported Codex message type %T", message)
		}
	}
	request.Instructions = codexInstructions(instructions)
	tools, err := buildCodexTools(c.Tools)
	if err != nil {
		return codexRequest{}, err
	}
	request.Tools = tools
	if len(tools) > 0 {
		request.ToolChoice = "auto"
		request.ParallelToolCalls = true
	}
	if opts != nil {
		request.MaxOutputTokens = opts.MaxTokens
		request.Temperature = opts.Temperature
		if opts.ReasoningEffort != "" {
			request.Reasoning = &codexReasoning{Effort: string(opts.ReasoningEffort), Summary: "auto"}
		}
		if opts.StructuredOutput != nil {
			spec := &StructuredOutputSpec{
				Name:       opts.StructuredOutput.Name,
				JSONSchema: append(json.RawMessage(nil), opts.StructuredOutput.JSONSchema...),
				Strict:     opts.StructuredOutput.Strict,
			}
			if err := applyCodexStructuredOutput(&request, spec); err != nil {
				return codexRequest{}, err
			}
		}
		if opts.CacheRetention != llm.CacheRetentionNone {
			request.PromptCacheKey = opts.SessionID
		}
		request.headers = opts.Headers
	}
	return request, nil
}

func newCodexRequest(model string) codexRequest {
	return codexRequest{
		Model:   model,
		Store:   false,
		Stream:  true,
		Text:    codexTextOptions{Verbosity: "low"},
		Include: []string{"reasoning.encrypted_content"},
	}
}

func codexInstructions(instructions []string) string {
	if len(instructions) == 0 {
		return "You are a helpful assistant."
	}
	return strings.Join(instructions, "\n\n")
}

func codexUserContent(text string, imageURLs []string) []any {
	content := make([]any, 0, 1+len(imageURLs))
	if text != "" {
		content = append(content, map[string]any{"type": "input_text", "text": text})
	}
	for _, imageURL := range imageURLs {
		if imageURL != "" {
			content = append(content, map[string]any{"type": "input_image", "image_url": imageURL, "detail": "auto"})
		}
	}
	return content
}

func codexNativeUserContent(content []llm.UserContent) ([]any, error) {
	projected := make([]any, 0, len(content))
	for _, block := range content {
		switch value := block.(type) {
		case llm.TextContent:
			projected = append(projected, map[string]any{"type": "input_text", "text": value.Text})
		case llm.ImageURLContent:
			projected = append(projected, map[string]any{"type": "input_image", "image_url": value.URL, "detail": "auto"})
		case llm.ImageContent:
			if _, err := base64.StdEncoding.DecodeString(value.Data); err != nil {
				return nil, errors.New("codex user message contains invalid image data")
			}
			projected = append(projected, map[string]any{
				"type":      "input_image",
				"image_url": "data:" + value.MimeType + ";base64," + value.Data,
				"detail":    "auto",
			})
		default:
			return nil, fmt.Errorf("unsupported Codex user content type %T", block)
		}
	}
	return projected, nil
}

func codexAssistantInput(text string) map[string]any {
	return map[string]any{
		"role": "assistant",
		"content": []any{
			map[string]any{"type": "input_text", "text": text},
		},
	}
}

func parseCodexReasoningSignature(signature string) (json.RawMessage, error) {
	if !json.Valid([]byte(signature)) {
		return nil, errors.New("codex reasoning signature is invalid JSON")
	}
	var item map[string]json.RawMessage
	if err := json.Unmarshal([]byte(signature), &item); err != nil || item == nil {
		return nil, errors.New("codex reasoning signature must be a JSON object")
	}
	var itemType string
	if err := json.Unmarshal(item["type"], &itemType); err != nil || itemType != "reasoning" {
		return nil, errors.New("codex reasoning signature must contain a reasoning output item")
	}
	var id string
	if err := json.Unmarshal(item["id"], &id); err != nil || id == "" {
		return nil, errors.New("codex reasoning signature must contain an item ID")
	}
	return append(json.RawMessage(nil), signature...), nil
}

func buildCodexTools(tools []llm.Tool) ([]codexTool, error) {
	projected := make([]codexTool, len(tools))
	for i, tool := range tools {
		var parameters map[string]any
		if err := json.Unmarshal(tool.Parameters, &parameters); err != nil || parameters == nil {
			return nil, fmt.Errorf("codex tool %q parameters must be a JSON object", tool.Name)
		}
		projected[i] = codexTool{
			Type:        "function",
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  append(json.RawMessage(nil), tool.Parameters...),
			Strict:      nil,
		}
	}
	return projected, nil
}

func applyCodexStructuredOutput(request *codexRequest, spec *StructuredOutputSpec) error {
	if spec == nil {
		return nil
	}
	if spec.Name == "" {
		return errors.New("structured output name is required")
	}
	if len(spec.JSONSchema) == 0 || !json.Valid(spec.JSONSchema) {
		return errors.New("structured output JSON schema is required")
	}
	var schema map[string]any
	if err := json.Unmarshal(spec.JSONSchema, &schema); err != nil || schema == nil {
		return errors.New("structured output JSON schema must be an object")
	}
	request.Text.Format = &codexResponseFormat{
		Type:   "json_schema",
		Name:   spec.Name,
		Schema: append(json.RawMessage(nil), spec.JSONSchema...),
		Strict: spec.Strict,
	}
	return nil
}

func parseCodexStream(reader io.Reader, onText func(string)) (codexResult, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	var dataLines []string
	result := codexResult{}
	toolArguments := make(map[int]string)
	outputItems := make(map[int]codexOutputItem)
	messageText := make(map[int]string)
	dispatch := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		data := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		if data == "[DONE]" {
			return nil
		}
		var event codexSSEEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return errors.New("decode Codex SSE event")
		}
		switch event.Type {
		case "response.output_text.delta", "response.refusal.delta":
			messageText[event.OutputIndex] += event.Delta
			if onText != nil && event.Delta != "" {
				onText(event.Delta)
			}
		case "response.output_item.added":
			item, err := parseCodexOutputItem(event.Item)
			if err != nil {
				return errors.New("decode Codex output item")
			}
			outputItems[event.OutputIndex] = item
			if item.parsed.Type == "function_call" {
				toolArguments[event.OutputIndex] = item.parsed.Arguments
			}
		case "response.function_call_arguments.delta":
			toolArguments[event.OutputIndex] += event.Delta
		case "response.function_call_arguments.done":
			toolArguments[event.OutputIndex] = event.Arguments
		case "response.output_item.done":
			item, err := parseCodexOutputItem(event.Item)
			if err != nil {
				return errors.New("decode Codex completed output item")
			}
			item.done = true
			outputItems[event.OutputIndex] = item
			switch item.parsed.Type {
			case "message":
				finalText := codexItemText(item.parsed)
				if err := recordCodexMessageText(messageText, event.OutputIndex, finalText, onText); err != nil {
					return err
				}
			case "function_call":
				if item.parsed.Arguments != "" {
					toolArguments[event.OutputIndex] = item.parsed.Arguments
				}
			}
		case "response.completed", "response.done", "response.incomplete":
			var terminal codexTerminalResponse
			if err := json.Unmarshal(event.Response, &terminal); err != nil {
				return errors.New("decode Codex terminal response")
			}
			if err := applyCodexTerminal(&result, terminal, outputItems, toolArguments, messageText, onText); err != nil {
				return err
			}
		case "response.failed":
			var terminal codexTerminalResponse
			_ = json.Unmarshal(event.Response, &terminal)
			if terminal.Error != nil && terminal.Error.Code != "" {
				return fmt.Errorf("codex response failed (%s)", terminal.Error.Code)
			}
			return errors.New("codex response failed")
		case "error":
			code := event.Code
			if code == "" && event.Error != nil {
				code = event.Error.Code
			}
			if code != "" {
				return fmt.Errorf("codex stream error (%s)", code)
			}
			return errors.New("codex stream error")
		}
		return nil
	}
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if err := dispatch(); err != nil {
				return codexResult{}, err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return codexResult{}, fmt.Errorf("read Codex SSE stream: %w", err)
	}
	if err := dispatch(); err != nil {
		return codexResult{}, err
	}
	if !result.terminal {
		return codexResult{}, errors.New("codex SSE stream ended before a terminal response")
	}
	return result, nil
}

func applyCodexTerminal(
	result *codexResult,
	terminal codexTerminalResponse,
	items map[int]codexOutputItem,
	arguments map[int]string,
	messageText map[int]string,
	onText func(string),
) error {
	result.id = terminal.ID
	result.model = terminal.Model
	result.status = terminal.Status
	result.cacheRead = terminal.Usage.InputDetails.CachedTokens
	result.cacheWrite = terminal.Usage.InputDetails.CacheWriteTokens
	result.input = max(0, terminal.Usage.InputTokens-result.cacheRead-result.cacheWrite)
	result.output = terminal.Usage.OutputTokens
	result.terminal = true
	for i, raw := range terminal.Output {
		terminalItem, err := parseCodexOutputItem(raw)
		if err != nil {
			return errors.New("decode Codex terminal output item")
		}
		if current, ok := items[i]; ok && current.done && current.parsed.Type == "reasoning" && terminalItem.parsed.Type == "reasoning" {
			terminalItem, err = backfillCodexReasoningItem(current, terminalItem)
			if err != nil {
				return err
			}
		}
		items[i] = terminalItem
		if terminalItem.parsed.Type == "message" {
			if err := recordCodexMessageText(messageText, i, codexItemText(terminalItem.parsed), onText); err != nil {
				return err
			}
		}
		if terminalItem.parsed.Type == "function_call" && terminalItem.parsed.Arguments != "" {
			arguments[i] = terminalItem.parsed.Arguments
		}
	}
	indexSet := make(map[int]struct{}, len(items)+len(messageText))
	for index := range items {
		indexSet[index] = struct{}{}
	}
	for index := range messageText {
		indexSet[index] = struct{}{}
	}
	indexes := make([]int, 0, len(indexSet))
	for index := range indexSet {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	result.text = ""
	for _, i := range indexes {
		item := items[i]
		switch item.parsed.Type {
		case "reasoning":
			signature, err := codexReasoningSignature(item)
			if err != nil {
				return err
			}
			result.content = append(result.content, llm.ThinkingContent{
				Thinking:  codexReasoningText(item.parsed),
				Signature: signature,
			})
			continue
		case "message":
			finalText := codexItemText(item.parsed)
			if finalText == "" {
				finalText = messageText[i]
			}
			result.text += finalText
			if finalText != "" {
				result.content = append(result.content, llm.TextContent{Text: finalText})
			}
			continue
		case "function_call":
		case "":
			result.text += messageText[i]
			continue
		default:
			continue
		}
		var decoded map[string]any
		encoded := arguments[i]
		if strings.TrimSpace(encoded) == "" {
			encoded = "{}"
		}
		if err := json.Unmarshal([]byte(encoded), &decoded); err != nil || decoded == nil {
			return fmt.Errorf("codex tool call %q returned invalid arguments", item.parsed.Name)
		}
		callID := item.parsed.CallID
		if callID == "" {
			callID = item.parsed.ID
		}
		if item.parsed.ID != "" && item.parsed.ID != callID {
			callID += "|" + item.parsed.ID
		}
		call := llm.ToolCall{
			ID:        callID,
			Name:      item.parsed.Name,
			Arguments: decoded,
		}
		result.toolCalls = append(result.toolCalls, call)
		result.content = append(result.content, call)
	}
	if terminal.Status == "failed" || terminal.Status == "cancelled" {
		return fmt.Errorf("codex response ended with status %s", terminal.Status)
	}
	return nil
}

func recordCodexMessageText(messageText map[int]string, index int, finalText string, onText func(string)) error {
	streamedText := messageText[index]
	if finalText == "" {
		return nil
	}
	if !strings.HasPrefix(finalText, streamedText) {
		return fmt.Errorf("codex message output %d does not match streamed text", index)
	}
	if onText != nil {
		if delta := strings.TrimPrefix(finalText, streamedText); delta != "" {
			onText(delta)
		}
	}
	messageText[index] = finalText
	return nil
}

func parseCodexOutputItem(raw json.RawMessage) (codexOutputItem, error) {
	if len(raw) == 0 || !json.Valid(raw) {
		return codexOutputItem{}, errors.New("invalid Codex output item")
	}
	var parsed codexResponseItem
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Type == "" {
		return codexOutputItem{}, errors.New("invalid Codex output item")
	}
	return codexOutputItem{
		parsed: parsed,
		raw:    append(json.RawMessage(nil), raw...),
	}, nil
}

func backfillCodexReasoningItem(streamed, terminal codexOutputItem) (codexOutputItem, error) {
	if streamed.parsed.EncryptedContent != "" || terminal.parsed.EncryptedContent == "" {
		return streamed, nil
	}
	var streamedFields, terminalFields map[string]json.RawMessage
	if err := json.Unmarshal(streamed.raw, &streamedFields); err != nil {
		return codexOutputItem{}, errors.New("decode streamed Codex reasoning item")
	}
	if err := json.Unmarshal(terminal.raw, &terminalFields); err != nil {
		return codexOutputItem{}, errors.New("decode terminal Codex reasoning item")
	}
	encryptedContent, ok := terminalFields["encrypted_content"]
	if !ok {
		return streamed, nil
	}
	streamedFields["encrypted_content"] = encryptedContent
	raw, err := json.Marshal(streamedFields)
	if err != nil {
		return codexOutputItem{}, errors.New("encode Codex reasoning signature")
	}
	backfilled, err := parseCodexOutputItem(raw)
	backfilled.done = true
	return backfilled, err
}

func codexReasoningSignature(item codexOutputItem) (string, error) {
	if item.parsed.Type != "reasoning" || item.parsed.ID == "" {
		return "", errors.New("codex reasoning output item is not replayable")
	}
	if item.parsed.EncryptedContent == "" {
		return "", nil
	}
	if _, err := parseCodexReasoningSignature(string(item.raw)); err != nil {
		return "", err
	}
	return string(item.raw), nil
}

func codexReasoningText(item codexResponseItem) string {
	var parts []string
	for _, summary := range item.Summary {
		if summary.Text != "" {
			parts = append(parts, summary.Text)
		}
	}
	if len(parts) == 0 {
		for _, content := range item.Content {
			if content.Text != "" {
				parts = append(parts, content.Text)
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

func codexItemText(item codexResponseItem) string {
	var text strings.Builder
	for _, content := range item.Content {
		switch content.Type {
		case "output_text":
			text.WriteString(content.Text)
		case "refusal":
			text.WriteString(content.Refusal)
		}
	}
	return text.String()
}

func (r codexResult) responseModel(requestModel string) string {
	if r.model != "" {
		return r.model
	}
	return requestModel
}

func codexResponsesURL(baseURL string) string {
	normalized := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(normalized, "/codex/responses") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/codex") {
		return normalized + "/responses"
	}
	return normalized + "/codex/responses"
}

func codexModelsURL(baseURL string) string {
	normalized := strings.TrimRight(baseURL, "/")
	normalized = strings.TrimSuffix(normalized, "/responses")
	if strings.HasSuffix(normalized, "/codex") {
		return normalized + "/models"
	}
	return normalized + "/codex/models"
}

func codexJWTMetadata(token string) (string, time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", time.Time{}, errors.New("failed to extract Codex account ID from access token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", time.Time{}, errors.New("failed to extract Codex account ID from access token")
	}
	var claims struct {
		Auth struct {
			AccountID string `json:"chatgpt_account_id"`
		} `json:"https://api.openai.com/auth"`
		ExpiresAt json.Number `json:"exp"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&claims); err != nil {
		return "", time.Time{}, errors.New("failed to decode Codex access token")
	}
	var expiresAt time.Time
	if claims.ExpiresAt != "" {
		seconds, err := claims.ExpiresAt.Int64()
		if err != nil {
			return "", time.Time{}, errors.New("invalid Codex access token expiry")
		}
		expiresAt = time.Unix(seconds, 0)
	}
	return claims.Auth.AccountID, expiresAt, nil
}
