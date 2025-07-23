package main

import (
	"fmt"
	"go/ast"
	"io"
	"iter"
	"log"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/gostaticanalysis/comment"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

type configType struct {
	Keys     []*configKey
	Comments []*ast.CommentGroup
}

type configKey struct {
	Name     string
	Type     string
	Required bool
	Default  string
	Comment  string
}

type decl struct {
	Decl   *ast.GenDecl
	Fields []*ast.Field
}

type entry[K comparable, V any] struct {
	Key   K
	Value V
}

func entries[K comparable, V any](iter iter.Seq2[K, V]) func(yield func(*entry[K, V]) bool) {
	return func(yield func(*entry[K, V]) bool) {
		for k, v := range iter {
			if !yield(&entry[K, V]{k, v}) {
				break
			}
		}
	}
}

func collectDecls(files []*ast.File) map[string]*decl {
	decls := make(map[string]*decl)
	for _, file := range files {
		for _, d := range file.Decls {
			genDecl, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.StructType); ok {
					decls[typeSpec.Name.Name] = &decl{
						Decl:   genDecl,
						Fields: typeSpec.Type.(*ast.StructType).Fields.List,
					}
				}
			}
		}
	}
	return decls
}

func collectConfigTypes(decls map[string]*decl, comments comment.Maps) map[string]*configType {
	configs := make(map[string]*configType)
	for name, decl := range decls {
		for i, field := range decl.Fields {
			if field.Tag == nil || field.Tag.Value == "" {
				continue
			}
			// strip the backticks and parse the tag
			tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
			key, ok := tag.Lookup("envconfig")
			if !ok {
				continue
			}
			if _, ok := configs[name]; !ok {
				configs[name] = &configType{
					Keys: []*configKey{},
				}
				d, ok := decls[name]
				if ok {
					c := comments.CommentsByPos(d.Decl.TokPos)
					configs[name].Comments = c
				}
			}
			configKey := &configKey{
				Name: key,
				Type: field.Type.(*ast.Ident).Name,
			}
			configs[name].Keys = append(configs[name].Keys, configKey)
			if required, ok := tag.Lookup("required"); ok {
				configKey.Required = required == "true"
			}
			if def, ok := tag.Lookup("default"); ok {
				configKey.Default = def
			}
			d, ok := decls[name]
			if ok {
				f := d.Fields[i]
				configKey.Comment = strings.ReplaceAll(f.Doc.Text(), "\n", "")
			}
		}
	}
	return configs
}

func loadPackages(packageName string) ([]*packages.Package, error) {
	return packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes,
		Dir:  packageName,
	})
}

func collectConfigTypesFromPackages(pkgs []*packages.Package) map[string]*configType {
	configs := map[string]*configType{}

	for _, pkg := range pkgs {
		decls := collectDecls(pkg.Syntax)
		comment := comment.New(pkg.Fset, pkg.Syntax)

		configInPkg := collectConfigTypes(decls, comment)
		maps.Copy(configs, configInPkg)
	}

	return configs
}

func writeMarkdown(w io.Writer, configs map[string]*configType) error {
	sortedEntries := slices.SortedFunc(entries(maps.All(configs)), func(a, b *entry[string, *configType]) int {
		return strings.Compare(a.Key, b.Key)
	})

	for _, entry := range sortedEntries {
		name := entry.Key
		config := entry.Value

		// write markdown
		fmt.Fprintf(w, "## %s\n\n", name)

		if len(config.Comments) > 0 {
			for _, c := range config.Comments {
				for _, line := range strings.Split(c.Text(), "\n") {
					fmt.Fprintf(w, "%s\n", line)
				}
			}
		}

		table := tablewriter.NewTable(w,
			tablewriter.WithRenderer(renderer.NewMarkdown()),
			tablewriter.WithConfig(tablewriter.NewConfigBuilder().
				Header().Alignment().WithGlobal(tw.AlignLeft).Build().
				Header().Formatting().WithAutoFormat(tw.Off).Build().Build().
				Build()),
		)

		table.Header([]string{"Name", "Type", "Required", "Default", "Comment"})
		for _, key := range config.Keys {
			defaults := ""
			if key.Default != "" {
				defaults = fmt.Sprintf("%q", key.Default)
			}
			err := table.Append(
				key.Name,
				key.Type,
				fmt.Sprintf("%t", key.Required),
				defaults,
				key.Comment,
			)
			if err != nil {
				return fmt.Errorf("failed to append row: %w", err)
			}
		}
		err := table.Render()
		if err != nil {
			return fmt.Errorf("failed to render table: %w", err)
		}

		fmt.Fprintln(w)
	}
	return nil
}

func main() {
	if err := newCommand().Execute(); err != nil {
		log.Fatalf("failed to execute command: %v", err)
	}
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Generate configuration documentation from Go source code",
		Long:  `This command generates markdown documentation for configuration structures annotated with envconfig tags.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgs, err := loadPackages(args[0])
			if err != nil {
				return fmt.Errorf("failed to load packages: %w", err)
			}
			configs := collectConfigTypesFromPackages(pkgs)
			return writeMarkdown(cmd.OutOrStdout(), configs)
		},
	}
	return cmd
}
