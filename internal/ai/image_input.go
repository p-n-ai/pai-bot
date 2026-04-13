// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

var supportedImageMIMETypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
	"image/gif":  {},
}

type normalizedImageInput struct {
	MIMEType string
	Data     []byte
	URL      string
}

func normalizeImageInput(raw string) (normalizedImageInput, error) {
	if strings.HasPrefix(raw, "data:") {
		return parseDataURLImage(raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return normalizedImageInput{}, fmt.Errorf("unsupported image input %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return normalizedImageInput{}, fmt.Errorf("unsupported image scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return normalizedImageInput{}, fmt.Errorf("unsupported image URL %q", raw)
	}

	return normalizedImageInput{URL: raw}, nil
}

func parseDataURLImage(raw string) (normalizedImageInput, error) {
	meta, payload, ok := strings.Cut(strings.TrimPrefix(raw, "data:"), ",")
	if !ok {
		return normalizedImageInput{}, fmt.Errorf("unsupported image data URL")
	}

	mediaType := meta
	mediaType = strings.TrimSuffix(mediaType, ";base64")
	if mediaType == "" {
		return normalizedImageInput{}, fmt.Errorf("unsupported image MIME type")
	}

	parsedMediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return normalizedImageInput{}, fmt.Errorf("unsupported image MIME type %q: %w", mediaType, err)
	}
	if !isSupportedImageMIMEType(parsedMediaType) {
		return normalizedImageInput{}, fmt.Errorf("unsupported image MIME type %q", parsedMediaType)
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return normalizedImageInput{}, fmt.Errorf("decode image data URL: %w", err)
	}

	return normalizedImageInput{
		MIMEType: parsedMediaType,
		Data:     data,
	}, nil
}

func fetchImageBytes(ctx context.Context, client *http.Client, rawURL string) (normalizedImageInput, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return normalizedImageInput{}, fmt.Errorf("create image request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return normalizedImageInput{}, fmt.Errorf("fetch image %q: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return normalizedImageInput{}, fmt.Errorf("fetch image %q: status %d", rawURL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return normalizedImageInput{}, fmt.Errorf("read image %q: %w", rawURL, err)
	}
	mimeType := normalizeImageMIMEType(resp.Header.Get("Content-Type"), data)
	if !isSupportedImageMIMEType(mimeType) {
		return normalizedImageInput{}, fmt.Errorf("unsupported image MIME type %q", mimeType)
	}

	return normalizedImageInput{
		MIMEType: mimeType,
		Data:     data,
		URL:      rawURL,
	}, nil
}

func normalizeImageMIMEType(contentType string, data []byte) string {
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil && mediaType != "" {
			return mediaType
		}
	}
	if len(data) == 0 {
		return ""
	}
	return http.DetectContentType(data)
}

func isSupportedImageMIMEType(mimeType string) bool {
	_, ok := supportedImageMIMETypes[mimeType]
	return ok
}
