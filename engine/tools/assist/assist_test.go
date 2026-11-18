package assist

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

const sample = `package main

import (
	"hexago/ui/geom"
	w "hexago/ui/widgets"
)

func build() w.Widget {
	return w.Column{Children: []w.Widget{
		w.Container{Color: red, Child: w.Text{Data: "a"}},
		w.Center{Child: w.Padding{Padding: geom.All(8), Child: w.SizedBox{}}},
	}}
}
`

var ws = Widgets{Engine: map[string]bool{
	"Column": true, "Container": true, "Text": true, "Center": true, "Padding": true, "SizedBox": true,
}}

// at — байтовая позиция первого вхождения s (курсор на начале слова).
func at(s string) int { return strings.Index(sample, s) }

// placeholder — место для ввода в сниппете: ${1:8} → 8, ${1} → пусто.
var placeholder = regexp.MustCompile(`\$\{\d+(?::([^}]*))?\}`)

// unescape превращает сниппет в обычный код (места для ввода — их значения по умолчанию),
// чтобы сравнивать его как текст.
func unescape(s string) string {
	s = placeholder.ReplaceAllString(s, "$1")
	return strings.NewReplacer(`\}`, "}", `\$`, "$", `\\`, `\`).Replace(s)
}

func ids(list []Assist) []string {
	var out []string
	for _, a := range list {
		out = append(out, a.ID)
	}
	return out
}

func apply(t *testing.T, cursor, id string) string {
	t.Helper()
	e, err := Apply([]byte(sample), at(cursor), id, ws)
	if err != nil {
		t.Fatal(err)
	}
	return unescape(e.Text)
}

func TestListForWidgetInColumn(t *testing.T) {
	got := ids(List([]byte(sample), at("w.Container{"), ws))
	for _, want := range []string{"wrap:Widget", "wrap:Center", "wrap:Expanded", "remove"} {
		if !slices.Contains(got, want) {
			t.Errorf("нет действия %s среди %v", want, got)
		}
	}
	if slices.Contains(got, "swap") {
		t.Error("swap недоступен: ребёнок Text — без своего Child")
	}
}

func TestInnermostWidget(t *testing.T) {
	e, err := Apply([]byte(sample), at("w.SizedBox{}"), "wrap:Center", ws)
	if err != nil {
		t.Fatal(err)
	}
	if got := unescape(e.Text); got != "w.Center{Child: w.SizedBox{}}" {
		t.Errorf("обёрнут не тот виджет: %q", got)
	}
}

func TestWrapInCenter(t *testing.T) {
	want := `w.Center{Child: w.Container{Color: red, Child: w.Text{Data: "a"}}}`
	if got := apply(t, "w.Container{", "wrap:Center"); got != want {
		t.Errorf("получили\n%s\nожидали\n%s", got, want)
	}
}

func TestWrapInPaddingHasPlaceholder(t *testing.T) {
	e, _ := Apply([]byte(sample), at("w.Container{"), "wrap:Padding", ws)
	if !strings.Contains(e.Text, "geom.All(${1:8})") {
		t.Errorf("в обёртке Padding нет места для ввода отступа: %s", e.Text)
	}
	if e.Before != nil {
		t.Errorf("geom уже импортирован, а правка добавляет импорт: %+v", e.Before)
	}
}

// applyEdit применяет правку так же, как расширение: сначала Before, потом основную.
func applyEdit(src string, e Edit) string {
	if b := e.Before; b != nil {
		src = src[:b.Start] + b.Text + src[b.End:]
	}
	return src[:e.Start] + unescape(e.Text) + src[e.End:]
}

func TestWrapAddsGeomImport(t *testing.T) {
	cases := map[string]struct{ src, want string }{
		"одиночный импорт": {
			"package main\n\nimport w \"hexago/ui/widgets\"\n\nvar x = w.SizedBox{}\n",
			"package main\n\nimport (\n\t\"hexago/ui/geom\"\n\tw \"hexago/ui/widgets\"\n)\n\nvar x = w.Padding{Padding: geom.All(8), Child: w.SizedBox{}}\n",
		},
		"группа импортов": {
			"package main\n\nimport (\n\t\"fmt\"\n\n\t\"hexago/ui/widgets\"\n)\n\nvar x = widgets.SizedBox{}\n",
			"package main\n\nimport (\n\t\"fmt\"\n\n\t\"hexago/ui/geom\"\n\t\"hexago/ui/widgets\"\n)\n\nvar x = widgets.Padding{Padding: geom.All(8), Child: widgets.SizedBox{}}\n",
		},
		"geom последним": {
			"package main\n\nimport (\n\t\"hexago/ui/app\"\n)\n\nimport \"hexago/ui/widgets\"\n\nvar x = widgets.SizedBox{}\n",
			"package main\n\nimport (\n\t\"hexago/ui/app\"\n\t\"hexago/ui/geom\"\n)\n\nimport \"hexago/ui/widgets\"\n\nvar x = widgets.Padding{Padding: geom.All(8), Child: widgets.SizedBox{}}\n",
		},
	}
	for name, c := range cases {
		e, err := Apply([]byte(c.src), strings.Index(c.src, "SizedBox"), "wrap:Padding", ws)
		if err != nil {
			t.Fatal(name, err)
		}
		if got := applyEdit(c.src, e); got != c.want {
			t.Errorf("%s: получили\n%s\nожидали\n%s", name, got, c.want)
		}
	}
}

func TestRemoveKeepsChild(t *testing.T) {
	if got := apply(t, "w.Container{", "remove"); got != `w.Text{Data: "a"}` {
		t.Errorf("после удаления осталось %q", got)
	}
}

func TestSwapWithChild(t *testing.T) {
	want := `w.Padding{Padding: geom.All(8), Child: w.Center{Child: w.SizedBox{}}}`
	if got := apply(t, "w.Center{", "swap"); got != want {
		t.Errorf("получили\n%s\nожидали\n%s", got, want)
	}
}

func TestNoAssistsOutsideWidgets(t *testing.T) {
	if got := List([]byte(sample), at("func build"), ws); len(got) != 0 {
		t.Errorf("вне виджета не должно быть действий, а есть %v", ids(got))
	}
}

func TestDotImportQualifier(t *testing.T) {
	src := "package main\n\nimport . \"hexago/ui/widgets\"\n\nvar x = Center{Child: SizedBox{}}\n"
	e, err := Apply([]byte(src), strings.Index(src, "SizedBox"), "wrap:Padding", ws)
	if err != nil {
		t.Fatal(err)
	}
	want := "package main\n\nimport (\n\t\"hexago/ui/geom\"\n\t. \"hexago/ui/widgets\"\n)\n\nvar x = Center{Child: Padding{Padding: geom.All(8), Child: SizedBox{}}}\n"
	if got := applyEdit(src, e); got != want {
		t.Errorf("при import . без префикса:\n%s", got)
	}
}
