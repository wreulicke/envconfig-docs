package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/go/packages"
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

func TestCollectConfigTypesFromPackages(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected map[string]*configType
	}{
		{
			name: "single config with envconfig tags",
			source: `
package test

// MyConfig is a test configuration
type MyConfig struct {
	// Database URL for connection
	DatabaseURL string ` + "`envconfig:\"DATABASE_URL\" required:\"true\" default:\"localhost:5432\"`" + `
	// API Key for authentication
	APIKey string ` + "`envconfig:\"API_KEY\" required:\"false\"`" + `
	// Max connections allowed
	MaxConnections int ` + "`envconfig:\"MAX_CONN\" default:\"10\"`" + `
}
`,
			expected: map[string]*configType{
				"MyConfig": {
					Keys: []*configKey{
						{
							Name:     "DATABASE_URL",
							Type:     "string",
							Required: true,
							Default:  "localhost:5432",
							Comment:  "Database URL for connection",
						},
						{
							Name:     "API_KEY",
							Type:     "string",
							Required: false,
							Default:  "",
							Comment:  "API Key for authentication",
						},
						{
							Name:     "MAX_CONN",
							Type:     "int",
							Required: false,
							Default:  "10",
							Comment:  "Max connections allowed",
						},
					},
				},
			},
		},
		{
			name: "multiple configs in same package",
			source: `
package test

type Config1 struct {
	Field1 string ` + "`envconfig:\"FIELD1\"`" + `
}

type Config2 struct {
	Field2 int ` + "`envconfig:\"FIELD2\" required:\"true\"`" + `
}
`,
			expected: map[string]*configType{
				"Config1": {
					Keys: []*configKey{
						{Name: "FIELD1", Type: "string", Required: false},
					},
				},
				"Config2": {
					Keys: []*configKey{
						{Name: "FIELD2", Type: "int", Required: true},
					},
				},
			},
		},
		{
			name: "struct without envconfig tags",
			source: `
package test

type NoEnvConfig struct {
	Field1 string
	Field2 int ` + "`json:\"field2\"`" + `
}
`,
			expected: map[string]*configType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the source code
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.source, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			// Create a mock package
			pkg := &packages.Package{
				Fset:   fset,
				Syntax: []*ast.File{file},
			}

			// Test the function
			result := collectConfigTypesFromPackages([]*packages.Package{pkg})

			// Compare results (ignoring Comments field for simplicity)
			for _, config := range result {
				config.Comments = nil
			}

			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("collectConfigTypesFromPackages() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCollectConfigTypesFromPackagesMultiplePackages(t *testing.T) {
	// Test with multiple packages
	source1 := `
package pkg1

type Config1 struct {
	Field1 string ` + "`envconfig:\"FIELD1\"`" + `
}
`
	source2 := `
package pkg2

type Config2 struct {
	Field2 string ` + "`envconfig:\"FIELD2\"`" + `
}
`

	fset := token.NewFileSet()
	file1, err := parser.ParseFile(fset, "pkg1.go", source1, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse source1: %v", err)
	}
	file2, err := parser.ParseFile(fset, "pkg2.go", source2, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse source2: %v", err)
	}

	pkg1 := &packages.Package{
		Fset:   fset,
		Syntax: []*ast.File{file1},
	}
	pkg2 := &packages.Package{
		Fset:   fset,
		Syntax: []*ast.File{file2},
	}

	result := collectConfigTypesFromPackages([]*packages.Package{pkg1, pkg2})

	expected := map[string]*configType{
		"Config1": {
			Keys: []*configKey{
				{Name: "FIELD1", Type: "string", Required: false},
			},
		},
		"Config2": {
			Keys: []*configKey{
				{Name: "FIELD2", Type: "string", Required: false},
			},
		},
	}

	// Ignore Comments field for comparison
	for _, config := range result {
		config.Comments = nil
	}

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("collectConfigTypesFromPackages() with multiple packages mismatch (-want +got):\n%s", diff)
	}
}
