package assist

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"hexago/tools/cli"
	"hexago/tools/sdk"
)

// Command — hexago assist: служебная команда для расширения VS Code.
//
//	hexago assist list  -file F -offset N [-stdin]          → JSON со списком действий
//	hexago assist apply -file F -offset N -id ID [-stdin]   → JSON с правкой
//
// offset — позиция курсора в байтах. С флагом -stdin текст файла читается со стандартного
// ввода: так учитываются несохранённые изменения в редакторе.
var Command = &cli.Command{
	Name:    "assist",
	Hidden:  true,
	Usage:   "assist list|apply -file <файл> -offset <байт> [-id <действие>] [-stdin]",
	Summary: "действия с виджетами для редактора (служебная команда расширения VS Code)",
	Help:    "Служебная команда расширения VS Code: список действий с виджетом под курсором и правки для них.",
	Run:     run,
}

func run(ctx *cli.Context, args []string) error {
	if len(args) == 0 || (args[0] != "list" && args[0] != "apply") {
		return cli.Usagef("нужно list или apply")
	}
	mode := args[0]
	fs := cli.NewFlagSet("assist")
	file := fs.String("file", "", "")
	offset := fs.Int("offset", -1, "")
	id := fs.String("id", "", "")
	stdin := fs.Bool("stdin", false, "")
	if _, err := cli.ParseArgs(fs, args[1:]); err != nil {
		return err
	}
	if *file == "" || *offset < 0 {
		return cli.Usagef("нужны -file и -offset")
	}

	var src []byte
	var err error
	if *stdin {
		src, err = io.ReadAll(os.Stdin)
	} else {
		src, err = os.ReadFile(*file)
	}
	if err != nil {
		return err
	}

	ws := Widgets{Engine: map[string]bool{}, Local: WidgetTypes(filepath.Dir(*file))}
	if engine, err := sdk.EnginePath(); err == nil {
		ws.Engine = WidgetTypes(filepath.Join(engine, "ui", "widgets"))
	}
	if f, _ := parser.ParseFile(token.NewFileSet(), "", src, parser.SkipObjectResolution); f != nil {
		addWidgetTypes(f, ws.Local) // виджеты из несохранённого текста файла
	}

	enc := json.NewEncoder(ctx.Stdout)
	if mode == "list" {
		list := List(src, *offset, ws)
		if list == nil {
			list = []Assist{}
		}
		return enc.Encode(list)
	}
	edit, err := Apply(src, *offset, *id, ws)
	if err != nil {
		return err
	}
	return enc.Encode(edit)
}

// WidgetTypes возвращает имена типов-виджетов (структур со встроенным Base) из папки пакета.
func WidgetTypes(dir string) map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if f, _ := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution); f != nil {
			addWidgetTypes(f, out)
		}
	}
	return out
}

func addWidgetTypes(f *ast.File, out map[string]bool) {
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts := spec.(*ast.TypeSpec)
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, fld := range st.Fields.List {
				if len(fld.Names) == 0 && typeName(fld.Type) == "Base" {
					out[ts.Name.Name] = true
				}
			}
		}
	}
}
