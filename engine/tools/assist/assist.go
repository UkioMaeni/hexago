// Package assist — действия с виджетами для редактора, как во Flutter:
// «Обернуть в виджет…», «Обернуть в Center/Padding/...», «Удалить виджет»,
// «Поменять местами с ребёнком».
//
// Расширение VS Code только показывает меню и применяет правку, а саму правку
// считает этот пакет по синтаксическому дереву Go — поэтому она точно попадает
// в нужные скобки, даже если внутри вложенные виджеты, строки и комментарии.
package assist

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// Assist — одно действие в меню редактора.
type Assist struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Edit — правка: заменить байты [Start, End) исходника на Text.
// Text может содержать места для ввода в синтаксисе сниппетов VS Code: ${1:8}.
type Edit struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
	// Before — обычная (не сниппет) правка, которую применить раньше основной: например,
	// импорт пакета geom для Padding{Padding: geom.All(8)}. Смещения основной правки уже
	// посчитаны для текста после неё.
	Before *Edit `json:"before,omitempty"`
}

// Widgets — имена типов-виджетов: из движка и из пакета, где редактируется файл.
type Widgets struct {
	Engine map[string]bool // типы пакета hexago/ui/widgets
	Local  map[string]bool // типы-виджеты, объявленные в пакете самого файла
}

// target — виджет под курсором и его окружение.
type target struct {
	src            []byte
	file           *ast.File
	fset           *token.FileSet
	lit            *ast.CompositeLit // сам виджет: Container{...}
	name           string            // имя типа: Container
	qual           string            // как в файле пишется пакет widgets: "w." или "" при import .
	inFlexChildren bool              // виджет — элемент Children у Row/Column/Flex
	inList         bool              // виджет — элемент любого списка []Widget{...}
}

// List возвращает действия, доступные для виджета под курсором (offset — в байтах).
func List(src []byte, offset int, ws Widgets) []Assist {
	t := find(src, offset, ws)
	if t == nil {
		return nil
	}
	var out []Assist
	for _, w := range wrappers {
		if w.id == "Expanded" && !t.inFlexChildren {
			continue
		}
		out = append(out, Assist{ID: "wrap:" + w.id, Title: w.title})
	}
	if _, ok := t.removal(); ok {
		out = append(out, Assist{ID: "remove", Title: "Удалить виджет (оставить содержимое)"})
	}
	if _, ok := t.swap(); ok {
		out = append(out, Assist{ID: "swap", Title: "Поменять местами с ребёнком"})
	}
	return out
}

// Apply вычисляет правку для действия id.
func Apply(src []byte, offset int, id string, ws Widgets) (Edit, error) {
	t := find(src, offset, ws)
	if t == nil {
		return Edit{}, fmt.Errorf("под курсором нет виджета")
	}
	start, end := t.span(t.lit)
	switch {
	case strings.HasPrefix(id, "wrap:"):
		name := strings.TrimPrefix(id, "wrap:")
		for _, w := range wrappers {
			if w.id == name {
				q := quals{widgets: t.qual, geom: "geom."}
				var before *Edit
				if w.geom {
					q.geom, before = importPackage(t.file, t.fset, src, "hexago/ui/geom", "geom")
				}
				e := Edit{Start: start, End: end, Text: w.wrap(q, escapeSnippet(string(src[start:end])))}
				if before != nil {
					shift := len(before.Text) - (before.End - before.Start)
					e.Start, e.End, e.Before = e.Start+shift, e.End+shift, before
				}
				return e, nil
			}
		}
		return Edit{}, fmt.Errorf("неизвестная обёртка %q", name)
	case id == "remove":
		text, ok := t.removal()
		if !ok {
			return Edit{}, fmt.Errorf("у виджета %s нет содержимого, которое можно оставить", t.name)
		}
		return Edit{Start: start, End: end, Text: escapeSnippet(text)}, nil
	case id == "swap":
		text, ok := t.swap()
		if !ok {
			return Edit{}, fmt.Errorf("поменять местами можно, только если ребёнок — виджет с полем Child")
		}
		return Edit{Start: start, End: end, Text: escapeSnippet(text)}, nil
	}
	return Edit{}, fmt.Errorf("неизвестное действие %q", id)
}

// quals — как в файле пишутся пакеты: "widgets.", "w.", "" при import . и т. п.
type quals struct{ widgets, geom string }

// wrapper — обёртка из меню.
type wrapper struct {
	id, title string
	geom      bool // нужен пакет hexago/ui/geom (отступы, скругления, выравнивания)
	wrap      func(q quals, child string) string
}

var wrappers = []wrapper{
	{"Widget", "Обернуть в виджет…", false, func(q quals, c string) string { return "${1:" + q.widgets + "Widget}{Child: " + c + "}" }},
	{"Center", "Обернуть в Center", false, func(q quals, c string) string { return q.widgets + "Center{Child: " + c + "}" }},
	{"Padding", "Обернуть в Padding", true, func(q quals, c string) string {
		return q.widgets + "Padding{Padding: " + q.geom + "All(${1:8}), Child: " + c + "}"
	}},
	{"Container", "Обернуть в Container", false, func(q quals, c string) string { return q.widgets + "Container{${1}Child: " + c + "}" }},
	{"SizedBox", "Обернуть в SizedBox", false, func(q quals, c string) string {
		return q.widgets + "SizedBox{Width: new(${1:100.0}), Height: new(${2:100.0}), Child: " + c + "}"
	}},
	{"Align", "Обернуть в Align", true, func(q quals, c string) string {
		return q.widgets + "Align{Alignment: " + q.geom + "${1:Center}, Child: " + c + "}"
	}},
	{"Expanded", "Обернуть в Expanded", false, func(q quals, c string) string { return q.widgets + "Expanded{Child: " + c + "}" }},
	{"Column", "Обернуть в Column", false, func(q quals, c string) string {
		return q.widgets + "Column{Children: []" + q.widgets + "Widget{" + c + "${1}}}"
	}},
	{"Row", "Обернуть в Row", false, func(q quals, c string) string {
		return q.widgets + "Row{Children: []" + q.widgets + "Widget{" + c + "${1}}}"
	}},
	{"Stack", "Обернуть в Stack", false, func(q quals, c string) string {
		return q.widgets + "Stack{Children: []" + q.widgets + "Widget{" + c + "${1}}}"
	}},
	{"Opacity", "Обернуть в Opacity", false, func(q quals, c string) string {
		return q.widgets + "Opacity{Opacity: ${1:0.5}, Child: " + c + "}"
	}},
	{"ClipRRect", "Обернуть в ClipRRect", true, func(q quals, c string) string {
		return q.widgets + "ClipRRect{BorderRadius: " + q.geom + "RadiusAll(${1:8}), Child: " + c + "}"
	}},
	{"SafeArea", "Обернуть в SafeArea", false, func(q quals, c string) string { return q.widgets + "SafeArea{Child: " + c + "}" }},
	{"Builder", "Обернуть в Builder", false, func(q quals, c string) string {
		return q.widgets + "Builder{Builder: func(ctx " + q.widgets + "BuildContext) " + q.widgets + "Widget {\n\treturn " + c + "\n}}"
	}},
}

// find разбирает файл и находит самый внутренний виджет, внутри которого стоит курсор.
func find(src []byte, offset int, ws Widgets) *target {
	fset := token.NewFileSet()
	f, _ := parser.ParseFile(fset, "", src, parser.ParseComments|parser.AllErrors)
	if f == nil {
		return nil
	}
	t := &target{src: src, file: f, fset: fset, qual: widgetsQualifier(f)}

	var stack []ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		stack = append(stack, n)
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		start, end := t.span(lit)
		if offset < start || offset > end {
			return true
		}
		if name, ok := t.widgetName(lit, ws); ok {
			// Внутренний виджет перезаписывает внешний: обход идёт сверху вниз.
			t.lit, t.name = lit, name
			t.inList, t.inFlexChildren = t.listContext(stack)
		}
		return true
	})
	if t.lit == nil {
		return nil
	}
	return t
}

// widgetName сообщает, что литерал — виджет, и возвращает имя его типа.
func (t *target) widgetName(lit *ast.CompositeLit, ws Widgets) (string, bool) {
	switch typ := lit.Type.(type) {
	case *ast.Ident: // Container{} при import . или виджет из своего пакета
		return typ.Name, ws.Local[typ.Name] || (t.qual == "" && ws.Engine[typ.Name])
	case *ast.SelectorExpr: // w.Container{}
		x, ok := typ.X.(*ast.Ident)
		return typ.Sel.Name, ok && x.Name+"." == t.qual && ws.Engine[typ.Sel.Name]
	}
	return "", false
}

// listContext определяет, лежит ли виджет в списке детей и чьих именно.
// stack заканчивается самим виджетом.
func (t *target) listContext(stack []ast.Node) (inList, inFlex bool) {
	if len(stack) < 4 {
		return false, false
	}
	list, ok := stack[len(stack)-2].(*ast.CompositeLit)
	if !ok {
		return false, false
	}
	if _, ok := list.Type.(*ast.ArrayType); !ok {
		return false, false
	}
	kv, ok := stack[len(stack)-3].(*ast.KeyValueExpr)
	if !ok {
		return true, false
	}
	parent, ok := stack[len(stack)-4].(*ast.CompositeLit)
	if !ok || keyName(kv) != "Children" {
		return true, false
	}
	switch typeName(parent.Type) {
	case "Row", "Column", "Flex":
		return true, true
	}
	return true, false
}

// field возвращает значение поля Key: литерала, например Child: X.
func field(lit *ast.CompositeLit, key string) (ast.Expr, bool) {
	for _, el := range lit.Elts {
		if kv, ok := el.(*ast.KeyValueExpr); ok && keyName(kv) == key {
			return kv.Value, true
		}
	}
	return nil, false
}

// removal — чем заменить виджет при удалении: его Child, единственным элементом Children
// или (если виджет в списке) всеми элементами Children.
func (t *target) removal() (string, bool) {
	if child, ok := field(t.lit, "Child"); ok {
		return t.text(child), true
	}
	children, ok := field(t.lit, "Children")
	if !ok {
		return "", false
	}
	list, ok := children.(*ast.CompositeLit)
	if !ok || len(list.Elts) == 0 {
		return "", false
	}
	if len(list.Elts) == 1 {
		return t.text(list.Elts[0]), true
	}
	if !t.inList {
		return "", false
	}
	parts := make([]string, len(list.Elts))
	for i, el := range list.Elts {
		parts[i] = t.text(el)
	}
	return strings.Join(parts, ",\n"), true
}

// swap меняет виджет местами с его ребёнком: X{.., Child: Y{.., Child: Z}} → Y{.., Child: X{.., Child: Z}}.
func (t *target) swap() (string, bool) {
	childExpr, ok := field(t.lit, "Child")
	if !ok {
		return "", false
	}
	y, ok := childExpr.(*ast.CompositeLit)
	if !ok {
		return "", false
	}
	grandchild, ok := field(y, "Child")
	if !ok {
		return "", false
	}
	// X' — X, у которого ребёнком стал Z.
	xs, xe := t.span(t.lit)
	cs, ce := t.span(childExpr)
	xPrime := string(t.src[xs:cs]) + t.text(grandchild) + string(t.src[ce:xe])
	// Y' — Y, у которого ребёнком стал X'.
	ys, ye := t.span(y)
	gs, ge := t.span(grandchild)
	return string(t.src[ys:gs]) + xPrime + string(t.src[ge:ye]), true
}

// span — байтовые границы узла в исходнике.
func (t *target) span(n ast.Node) (int, int) {
	return t.fset.Position(n.Pos()).Offset, t.fset.Position(n.End()).Offset
}

func (t *target) text(n ast.Node) string {
	s, e := t.span(n)
	return string(t.src[s:e])
}

func keyName(kv *ast.KeyValueExpr) string {
	if id, ok := kv.Key.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

// widgetsQualifier — как в файле обращаются к пакету hexago/ui/widgets:
// "w." (import w "..."), "widgets." (обычный импорт) или "" (import . "...").
func widgetsQualifier(f *ast.File) string {
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path != "hexago/ui/widgets" {
			continue
		}
		if imp.Name == nil {
			return "widgets."
		}
		if imp.Name.Name == "." {
			return ""
		}
		return imp.Name.Name + "."
	}
	return "widgets."
}

// escapeSnippet экранирует символы, особые в сниппетах VS Code ($, }, \),
// чтобы исходный код вставился как есть.
func escapeSnippet(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `$`, `\$`, `}`, `\}`)
	return r.Replace(s)
}

// importPackage возвращает, как в файле обращаться к пакету path ("geom.", "g.", "" при import .),
// и правку, которая добавит импорт, если его ещё нет (иначе nil). name — имя пакета по умолчанию.
// Новый импорт встаёт по алфавиту среди остальных, как его поставил бы goimports.
func importPackage(f *ast.File, fset *token.FileSet, src []byte, path, name string) (string, *Edit) {
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == path {
			switch {
			case imp.Name == nil:
				return name + ".", nil
			case imp.Name.Name == ".":
				return "", nil
			default:
				return imp.Name.Name + ".", nil
			}
		}
	}
	off := func(p token.Pos) int { return fset.Position(p).Offset }
	quoted := strconv.Quote(path)
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.IMPORT {
			continue
		}
		if !g.Lparen.IsValid() {
			// import "x" → import (\n\t"x"\n\t"path"\n), по алфавиту.
			lines := []string{string(src[off(g.Specs[0].Pos()):off(g.Specs[0].End())]), quoted}
			if importPath(g.Specs[0]) > path {
				lines[0], lines[1] = lines[1], lines[0]
			}
			text := "import (\n\t" + strings.Join(lines, "\n\t") + "\n)"
			return name + ".", &Edit{Start: off(g.Pos()), End: off(g.End()), Text: text}
		}
		for _, s := range g.Specs {
			if importPath(s) > path {
				at := off(s.Pos())
				return name + ".", &Edit{Start: at, End: at, Text: quoted + "\n\t"}
			}
		}
		at := off(g.Specs[len(g.Specs)-1].End())
		return name + ".", &Edit{Start: at, End: at, Text: "\n\t" + quoted}
	}
	// Импортов нет вовсе — сразу после строки package.
	at := off(f.Name.End())
	return name + ".", &Edit{Start: at, End: at, Text: "\n\nimport " + quoted}
}

// importPath — путь пакета в строке импорта.
func importPath(s ast.Spec) string {
	p, _ := strconv.Unquote(s.(*ast.ImportSpec).Path.Value)
	return p
}
