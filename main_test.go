package main

import (
	"bytes"
	"go/ast"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriteMarkdown(t *testing.T) {
	configs := map[string]*configType{
		"TestConfig": {
			Keys: []*configKey{
				{Name: "Key1", Type: "string", Required: true, Default: "default1", Comment: "This is key 1"},
				{Name: "Key2", Type: "int", Required: false, Default: "0", Comment: "This is key 2"},
			},
			Comments: []*ast.CommentGroup{
				{List: []*ast.Comment{{Text: "// This is a test config"}}},
			},
		},
	}

	var buf bytes.Buffer
	if err := writeMarkdown(&buf, configs); err != nil {
		t.Fatalf("writeMarkdown failed: %v", err)
	}

	expected := `## TestConfig

This is a test config

| Name | Type   | Required | Default    | Comment       |
|:-----|:-------|:---------|:-----------|:--------------|
| Key1 | string | true     | "default1" | This is key 1 |
| Key2 | int    | false    | "0"        | This is key 2 |

`
	if diff := cmp.Diff(buf.String(), expected); diff != "" {
		t.Errorf("writeMarkdown output did not match expected:\n%s", diff)
	}
}
