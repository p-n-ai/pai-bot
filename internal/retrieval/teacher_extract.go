// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retrieval

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	pdf "github.com/Detective-XH/gopdf"
)

const (
	MaxTeacherResourceBytes = 20 << 20
	maxArchiveEntries       = 2048
	maxArchiveEntryBytes    = 8 << 20
	maxArchiveTotalBytes    = 64 << 20
	chunkTargetBytes        = 1600
	chunkOverlapBytes       = 240
)

var (
	ErrUnsupportedFile = errors.New("unsupported teacher resource file")
	ErrMalformedFile   = errors.New("malformed teacher resource file")
	ErrEncryptedFile   = errors.New("encrypted teacher resource file")
	ErrImageOnlyFile   = errors.New("image-only teacher resource file")
	ErrEmptyFile       = errors.New("empty teacher resource file")
	ErrFileTooLarge    = errors.New("teacher resource exceeds 20 MiB")
	slideNamePattern   = regexp.MustCompile(`^ppt/slides/slide([0-9]+)\.xml$`)
)

type ExtractedUnit struct {
	LocatorType  string
	LocatorStart int
	LocatorEnd   int
	Text         string
}

type TeacherChunk struct {
	Ordinal      int
	LocatorType  string
	LocatorStart int
	LocatorEnd   int
	Content      string
}

type TeacherChunkEdge struct {
	SourceOrdinal int
	TargetOrdinal int
	Type          string
}

func ExtractTeacherResource(filename string, data []byte) (string, []ExtractedUnit, error) {
	if len(data) == 0 {
		return "", nil, ErrEmptyFile
	}
	if len(data) > MaxTeacherResourceBytes {
		return "", nil, ErrFileTooLarge
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		units, err := extractPDF(data)
		return "pdf", units, err
	case ".docx":
		units, err := extractOffice(data, "docx")
		return "docx", units, err
	case ".pptx":
		units, err := extractOffice(data, "pptx")
		return "pptx", units, err
	default:
		return "", nil, ErrUnsupportedFile
	}
}

func extractPDF(data []byte) ([]ExtractedUnit, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "encrypt") || strings.Contains(lower, "password") {
			return nil, fmt.Errorf("%w: PDF must not be password protected", ErrEncryptedFile)
		}
		return nil, fmt.Errorf("%w: PDF: %v", ErrMalformedFile, err)
	}
	summary := reader.DocumentSummary()
	if summary.TotalPages <= 0 || len(summary.Pages) == 0 {
		return nil, fmt.Errorf("%w: PDF has no readable pages", ErrEmptyFile)
	}
	if summary.ImageOnlyPages > 0 && summary.TextPages == 0 {
		return nil, fmt.Errorf("%w: PDF requires OCR", ErrImageOnlyFile)
	}
	if summary.DegradedPages > 0 {
		return nil, fmt.Errorf("%w: PDF text extraction diagnostics marked %d page(s) degraded", ErrMalformedFile, summary.DegradedPages)
	}
	units := make([]ExtractedUnit, 0, summary.TextPages)
	for pageNumber := 1; pageNumber <= reader.NumPage(); pageNumber++ {
		text, pageErr := reader.Page(pageNumber).GetPlainText(nil)
		if pageErr != nil {
			return nil, fmt.Errorf("%w: PDF page %d: %v", ErrMalformedFile, pageNumber, pageErr)
		}
		text = normalizeExtractedText(text)
		if text != "" {
			units = append(units, ExtractedUnit{
				LocatorType: "page", LocatorStart: pageNumber, LocatorEnd: pageNumber, Text: text,
			})
		}
	}
	if len(units) == 0 {
		return nil, fmt.Errorf("%w: PDF contains no extractable text", ErrEmptyFile)
	}
	return units, nil
}

func extractOffice(data []byte, kind string) ([]ExtractedUnit, error) {
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}) {
		return nil, fmt.Errorf("%w: encrypted Office documents are not supported", ErrEncryptedFile)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("%w: %s archive: %v", ErrMalformedFile, kind, err)
	}
	if len(archive.File) > maxArchiveEntries {
		return nil, fmt.Errorf("%w: archive has too many entries", ErrMalformedFile)
	}
	files := make(map[string]*zip.File, len(archive.File))
	var total uint64
	hasMedia := false
	for _, file := range archive.File {
		if file.UncompressedSize64 > maxArchiveEntryBytes {
			return nil, fmt.Errorf("%w: archive entry is too large", ErrMalformedFile)
		}
		total += file.UncompressedSize64
		if total > maxArchiveTotalBytes {
			return nil, fmt.Errorf("%w: archive expands beyond limit", ErrMalformedFile)
		}
		if file.CompressedSize64 > 0 && file.UncompressedSize64/file.CompressedSize64 > 100 {
			return nil, fmt.Errorf("%w: suspicious archive compression ratio", ErrMalformedFile)
		}
		clean := filepath.ToSlash(filepath.Clean(file.Name))
		if strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return nil, fmt.Errorf("%w: unsafe archive path", ErrMalformedFile)
		}
		lowerName := strings.ToLower(clean)
		if strings.HasSuffix(lowerName, "vbaproject.bin") {
			return nil, fmt.Errorf("%w: macro-enabled Office files are not supported", ErrUnsupportedFile)
		}
		if strings.HasPrefix(lowerName, "word/media/") || strings.HasPrefix(lowerName, "ppt/media/") {
			hasMedia = true
		}
		files[clean] = file
	}

	if kind == "docx" {
		file := files["word/document.xml"]
		if file == nil {
			return nil, fmt.Errorf("%w: DOCX document.xml missing", ErrMalformedFile)
		}
		text, err := extractXMLText(file)
		if err != nil {
			return nil, err
		}
		if text == "" {
			if hasMedia {
				return nil, fmt.Errorf("%w: DOCX requires OCR", ErrImageOnlyFile)
			}
			return nil, fmt.Errorf("%w: DOCX contains no text", ErrEmptyFile)
		}
		return []ExtractedUnit{{LocatorType: "section", LocatorStart: 1, LocatorEnd: 1, Text: text}}, nil
	}

	type slideFile struct {
		number int
		file   *zip.File
	}
	var slides []slideFile
	for name, file := range files {
		match := slideNamePattern.FindStringSubmatch(name)
		if len(match) != 2 {
			continue
		}
		number, _ := strconv.Atoi(match[1])
		slides = append(slides, slideFile{number: number, file: file})
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].number < slides[j].number })
	if len(slides) == 0 {
		return nil, fmt.Errorf("%w: PPTX contains no slides", ErrMalformedFile)
	}
	units := make([]ExtractedUnit, 0, len(slides))
	for _, slide := range slides {
		text, err := extractXMLText(slide.file)
		if err != nil {
			return nil, err
		}
		if text != "" {
			units = append(units, ExtractedUnit{
				LocatorType: "slide", LocatorStart: slide.number, LocatorEnd: slide.number, Text: text,
			})
		}
	}
	if len(units) == 0 {
		if hasMedia {
			return nil, fmt.Errorf("%w: PPTX requires OCR", ErrImageOnlyFile)
		}
		return nil, fmt.Errorf("%w: PPTX contains no text", ErrEmptyFile)
	}
	return units, nil
}

func extractXMLText(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("%w: open XML: %v", ErrMalformedFile, err)
	}
	defer rc.Close()
	raw, err := io.ReadAll(io.LimitReader(rc, maxArchiveEntryBytes+1))
	if err != nil || len(raw) > maxArchiveEntryBytes {
		return "", fmt.Errorf("%w: read XML entry", ErrMalformedFile)
	}
	upper := bytes.ToUpper(raw)
	if bytes.Contains(upper, []byte("<!DOCTYPE")) || bytes.Contains(upper, []byte("<!ENTITY")) {
		return "", fmt.Errorf("%w: XML declarations are not allowed", ErrMalformedFile)
	}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	decoder.Strict = true
	var parts []string
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("%w: XML: %v", ErrMalformedFile, err)
		}
		charData, ok := token.(xml.CharData)
		if ok {
			text := normalizeExtractedText(string(charData))
			if text != "" {
				parts = append(parts, text)
			}
		}
	}
	return normalizeExtractedText(strings.Join(parts, " ")), nil
}

func ChunkTeacherResource(units []ExtractedUnit) []TeacherChunk {
	var chunks []TeacherChunk
	for _, unit := range units {
		words := strings.Fields(unit.Text)
		for start := 0; start < len(words); {
			end := start
			size := 0
			for end < len(words) {
				next := len(words[end])
				if end > start {
					next++
				}
				if size+next > chunkTargetBytes && end > start {
					break
				}
				size += next
				end++
			}
			content := strings.Join(words[start:end], " ")
			chunks = append(chunks, TeacherChunk{
				Ordinal: len(chunks), LocatorType: unit.LocatorType,
				LocatorStart: unit.LocatorStart, LocatorEnd: unit.LocatorEnd, Content: content,
			})
			if end == len(words) {
				break
			}
			overlap := 0
			nextStart := end
			for nextStart > start && overlap < chunkOverlapBytes {
				nextStart--
				overlap += len(words[nextStart]) + 1
			}
			if nextStart <= start {
				nextStart = end
			}
			start = nextStart
		}
	}
	return chunks
}

func RelatedTeacherChunkEdges(chunks []TeacherChunk) []TeacherChunkEdge {
	lastByTerm := map[string]int{}
	seenPairs := map[[2]int]struct{}{}
	edges := make([]TeacherChunkEdge, 0, len(chunks))
	for ordinal, chunk := range chunks {
		terms := map[string]struct{}{}
		for _, raw := range strings.Fields(strings.ToLower(chunk.Content)) {
			term := strings.Trim(raw, ".,;:!?()[]{}\"'")
			if len(term) >= 5 {
				terms[term] = struct{}{}
			}
		}
		candidates := map[int]int{}
		for term := range terms {
			if previous, ok := lastByTerm[term]; ok && ordinal-previous > 1 {
				candidates[previous]++
			}
			lastByTerm[term] = ordinal
		}
		type candidate struct {
			ordinal int
			shared  int
		}
		ranked := make([]candidate, 0, len(candidates))
		for previous, shared := range candidates {
			ranked = append(ranked, candidate{ordinal: previous, shared: shared})
		}
		sort.Slice(ranked, func(i, j int) bool {
			if ranked[i].shared == ranked[j].shared {
				return ranked[i].ordinal < ranked[j].ordinal
			}
			return ranked[i].shared > ranked[j].shared
		})
		if len(ranked) > 2 {
			ranked = ranked[:2]
		}
		for _, match := range ranked {
			pair := [2]int{match.ordinal, ordinal}
			if _, exists := seenPairs[pair]; exists {
				continue
			}
			seenPairs[pair] = struct{}{}
			edges = append(edges, TeacherChunkEdge{
				SourceOrdinal: match.ordinal, TargetOrdinal: ordinal, Type: "related",
			})
		}
	}
	return edges
}

func normalizeExtractedText(value string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(value, "\x00", "")), " ")
}
