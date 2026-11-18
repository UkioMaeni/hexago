package widgets_test

import (
	"fmt"
	"strings"
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	w "hexago/ui/widgets"
)

// counter — виджет с состоянием: Title — параметр от родителя, count — своё состояние.
type counter struct {
	w.State
	Title string

	count    int
	log      *[]string // куда писать вызовы жизненного цикла (общий для теста)
	oldTitle string
}

func (c *counter) Init(w.BuildContext) {
	c.count = 10
	*c.log = append(*c.log, "Init")
}

func (c *counter) Update(_ w.BuildContext, old counter) {
	c.oldTitle = old.Title
	*c.log = append(*c.log, "Update")
}

func (c *counter) Dispose() { *c.log = append(*c.log, "Dispose") }

func (c *counter) Build(w.BuildContext) w.Widget {
	*c.log = append(*c.log, fmt.Sprintf("Build %s %d", c.Title, c.count))
	return w.SizedBox{}
}

// findCounter — живой экземпляр counter в дереве.
func findCounter(e *w.Element) *counter {
	if c, ok := e.State().(*counter); ok {
		return c
	}
	for _, ch := range e.Children() {
		if c := findCounter(ch); c != nil {
			return c
		}
	}
	return nil
}

func drawFrame(b *w.Binding) {
	b.DrawFrame(gfx.NewImageCanvas(10, 10), w.MediaQueryData{Size: geom.Size{Width: 10, Height: 10}, DevicePixelRatio: 1})
}

func TestStateLivesBetweenRebuilds(t *testing.T) {
	var log []string
	b := w.NewBinding(w.Center{Child: counter{Title: "a", log: &log}})
	drawFrame(b)
	c := findCounter(b.Root())
	if c == nil || c.count != 10 {
		t.Fatalf("Init не задал состояние: %+v", c)
	}

	// SetState меняет состояние и перестраивает виджет.
	c.SetState(func() { c.count++ })
	drawFrame(b)

	// Родитель перестроился с новым параметром — состояние осталось, параметр обновился.
	b.SetApp(w.Center{Child: counter{Title: "b", log: &log}})
	drawFrame(b)
	if got := findCounter(b.Root()); got != c {
		t.Fatal("после перестройки родителя живой экземпляр пересоздан")
	}
	if c.Title != "b" || c.count != 11 || c.oldTitle != "a" {
		t.Errorf("Title=%q count=%d oldTitle=%q; ожидали b, 11, a", c.Title, c.count, c.oldTitle)
	}

	// Виджет убрали из дерева — Dispose, и SetState больше нельзя.
	b.SetApp(w.Center{})
	drawFrame(b)
	if c.Mounted() {
		t.Error("после удаления из дерева Mounted() = true")
	}

	want := "Init, Build a 10, Build a 11, Update, Build b 11, Dispose"
	if got := strings.Join(log, ", "); got != want {
		t.Errorf("жизненный цикл:\n%s\nожидали\n%s", got, want)
	}
}

// valueCounter — ошибка: методы виджета с состоянием объявлены у значения.
type valueCounter struct {
	w.State
	count int
}

func (c valueCounter) Build(w.BuildContext) w.Widget { return w.SizedBox{} }

// badUpdate — ошибка: Update с неправильной сигнатурой.
type badUpdate struct{ w.State }

func (c *badUpdate) Build(w.BuildContext) w.Widget { return w.SizedBox{} }
func (c *badUpdate) Update(old badUpdate)          {}

func TestStatefulMistakesPanicClearly(t *testing.T) {
	cases := map[string]struct {
		widget w.Widget
		want   string
	}{
		"методы у значения": {valueCounter{}, "объявляются у указателя"},
		"неверный Update":   {badUpdate{}, "Update у badUpdate должен быть таким"},
		"передан указатель": {&counter{}, "передают значением"},
		"нет метода Build":  {struct{ w.State }{}, "нет метода Build"},
	}
	for name, c := range cases {
		func() {
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(fmt.Sprint(r), c.want) {
					t.Errorf("%s: паника %v, ожидали текст %q", name, r, c.want)
				}
			}()
			new(w.BuildOwner).Mount(c.widget)
		}()
	}
}
