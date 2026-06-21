package internal

import "testing"

func TestLooksQuestionMarkCorrupted(t *testing.T) {
	if !looksQuestionMarkCorrupted("????????????????") {
		t.Fatalf("expected question-mark-only text to be treated as corrupted")
	}
	if looksQuestionMarkCorrupted("你好？？") {
		t.Fatalf("short normal text with punctuation should not be treated as corrupted")
	}
	if looksQuestionMarkCorrupted("What is this?") {
		t.Fatalf("normal English question should not be treated as corrupted")
	}
}

func TestParseQwenImageResourceString(t *testing.T) {
	resource, ok := parseQwenImageResourceString(`{"metadata":{"qianwen_material_id":"mat-1","qwen_resource":{"id":"mat-1","url":"https://example.com/a.png","width":720,"height":1280}}}`)
	if !ok {
		t.Fatalf("expected qwen image resource")
	}
	if resource.ID != "mat-1" || resource.URL != "https://example.com/a.png" {
		t.Fatalf("unexpected resource: %+v", resource)
	}
}

func TestQwenVideoAttachments(t *testing.T) {
	attachments := qwenVideoAttachments([]qwenImageResource{{ID: "mat-1", URL: "https://example.com/a.png"}})
	if len(attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(attachments))
	}
	if attachments[0]["type"] != "image" || attachments[0]["materialId"] != "mat-1" {
		t.Fatalf("unexpected attachment: %+v", attachments[0])
	}
}
