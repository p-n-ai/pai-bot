// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package retrieval

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestExtractTeacherResourceFormats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		filename    string
		data        []byte
		wantType    string
		wantLocator string
		wantText    string
	}{
		{
			name: "pdf", filename: "lesson.pdf", data: generatedTextPDF("Quadratic equations"),
			wantType: "pdf", wantLocator: "page", wantText: "Quadratic equations",
		},
		{
			name: "docx", filename: "lesson.docx",
			data: officeArchive(t, map[string]string{
				"[Content_Types].xml": `<?xml version="1.0"?><Types/>`,
				"word/document.xml":   `<?xml version="1.0"?><w:document xmlns:w="w"><w:body><w:p><w:r><w:t>Linear equations</w:t></w:r></w:p></w:body></w:document>`,
			}),
			wantType: "docx", wantLocator: "section", wantText: "Linear equations",
		},
		{
			name: "pptx", filename: "lesson.pptx",
			data: officeArchive(t, map[string]string{
				"[Content_Types].xml":   `<?xml version="1.0"?><Types/>`,
				"ppt/slides/slide2.xml": `<p:sld xmlns:p="p" xmlns:a="a"><a:t>Second slide</a:t></p:sld>`,
				"ppt/slides/slide1.xml": `<p:sld xmlns:p="p" xmlns:a="a"><a:t>First slide</a:t></p:sld>`,
			}),
			wantType: "pptx", wantLocator: "slide", wantText: "First slide",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sourceType, units, err := ExtractTeacherResource(test.filename, test.data)
			if err != nil {
				t.Fatalf("ExtractTeacherResource() error = %v", err)
			}
			if sourceType != test.wantType || len(units) == 0 {
				t.Fatalf("type, units = %q, %#v", sourceType, units)
			}
			if units[0].LocatorType != test.wantLocator || units[0].Text != test.wantText {
				t.Fatalf("unit = %#v", units[0])
			}
		})
	}
}

func TestExtractTeacherResourceRejectsMalformedAndLimits(t *testing.T) {
	t.Parallel()
	if _, _, err := ExtractTeacherResource("bad.docx", []byte("not a zip")); !errors.Is(err, ErrMalformedFile) {
		t.Fatalf("malformed DOCX error = %v", err)
	}
	malformedXML := officeArchive(t, map[string]string{"word/document.xml": `<w:document><w:t>broken`})
	if _, _, err := ExtractTeacherResource("bad.docx", malformedXML); !errors.Is(err, ErrMalformedFile) {
		t.Fatalf("malformed XML error = %v", err)
	}
	zipBomb := officeArchive(t, map[string]string{"word/document.xml": string(bytes.Repeat([]byte("a"), 2<<20))})
	if _, _, err := ExtractTeacherResource("bomb.docx", zipBomb); !errors.Is(err, ErrMalformedFile) {
		t.Fatalf("zip bomb error = %v", err)
	}
	if _, _, err := ExtractTeacherResource("large.pdf", make([]byte, MaxTeacherResourceBytes+1)); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("oversize error = %v", err)
	}
	if _, _, err := ExtractTeacherResource("lesson.txt", []byte("text")); !errors.Is(err, ErrUnsupportedFile) {
		t.Fatalf("unsupported error = %v", err)
	}
	macro := officeArchive(t, map[string]string{
		"word/document.xml":   `<w:document xmlns:w="w"><w:t>text</w:t></w:document>`,
		"word/vbaProject.bin": "macro",
	})
	if _, _, err := ExtractTeacherResource("macro.docx", macro); !errors.Is(err, ErrUnsupportedFile) {
		t.Fatalf("macro error = %v", err)
	}
	imageOnly := officeArchive(t, map[string]string{
		"word/document.xml":    `<w:document xmlns:w="w"><w:body/></w:document>`,
		"word/media/image.png": "image",
	})
	if _, _, err := ExtractTeacherResource("scan.docx", imageOnly); !errors.Is(err, ErrImageOnlyFile) {
		t.Fatalf("image-only error = %v", err)
	}
	encryptedOffice := []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}
	if _, _, err := ExtractTeacherResource("encrypted.docx", encryptedOffice); !errors.Is(err, ErrEncryptedFile) {
		t.Fatalf("encrypted Office error = %v", err)
	}
}

func TestChunkTeacherResourceIsDeterministicWithOverlap(t *testing.T) {
	t.Parallel()
	text := bytes.Repeat([]byte("algebraic-expression "), 200)
	units := []ExtractedUnit{{LocatorType: "page", LocatorStart: 3, LocatorEnd: 3, Text: string(text)}}
	first := ChunkTeacherResource(units)
	second := ChunkTeacherResource(units)
	if fmt.Sprint(first) != fmt.Sprint(second) || len(first) < 2 {
		t.Fatalf("chunks are not deterministic: %d, %d", len(first), len(second))
	}
	if first[0].LocatorStart != 3 || first[1].LocatorStart != 3 {
		t.Fatalf("locators not preserved: %#v", first)
	}
	firstWords := bytes.Fields([]byte(first[0].Content))
	secondWords := bytes.Fields([]byte(first[1].Content))
	if len(firstWords) == 0 || len(secondWords) == 0 || !bytes.Equal(firstWords[len(firstWords)-1], secondWords[0]) {
		t.Fatal("expected deterministic chunk overlap")
	}
}

func TestRelatedTeacherChunkEdgesAreBoundedAndDeterministic(t *testing.T) {
	t.Parallel()
	chunks := []TeacherChunk{
		{Ordinal: 0, Content: "quadratic factorisation method"},
		{Ordinal: 1, Content: "unrelated geometry lesson"},
		{Ordinal: 2, Content: "quadratic factorisation practice"},
		{Ordinal: 3, Content: "another unrelated topic"},
		{Ordinal: 4, Content: "quadratic factorisation review"},
	}
	first := RelatedTeacherChunkEdges(chunks)
	second := RelatedTeacherChunkEdges(chunks)
	if fmt.Sprint(first) != fmt.Sprint(second) || len(first) == 0 {
		t.Fatalf("related edges = %#v, %#v", first, second)
	}
	for _, edge := range first {
		if edge.TargetOrdinal-edge.SourceOrdinal <= 1 || edge.Type != "related" {
			t.Fatalf("invalid related edge = %#v", edge)
		}
	}
}

func officeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func generatedTextPDF(text string) []byte {
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	offsets := make([]int, 6)
	writeObject := func(number int, body string) {
		offsets[number] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", number, body)
	}
	writeObject(1, "<< /Type /Catalog /Pages 2 0 R >>")
	writeObject(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	writeObject(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>")
	writeObject(4, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	stream := fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", text)
	writeObject(5, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	xref := output.Len()
	output.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xref)
	return output.Bytes()
}
