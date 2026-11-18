package widgets_test

import (
	"testing"

	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/hexagotest"
	"hexago/ui/painting"
	"hexago/ui/rendering"
	w "hexago/ui/widgets"
)

// layoutScene — экран из основных виджетов раскладки.
func layoutScene() w.Widget {
	chip := func(width float64, col gfx.Color) w.Widget {
		return w.Container{Width: new(width), Height: new(28.0),
			Decoration: &painting.BoxDecoration{Color: col, BorderRadius: geom.RadiusAll(14)}}
	}
	circle := func(col gfx.Color) w.Widget {
		return w.Container{Width: new(48.0), Height: new(48.0),
			Decoration: &painting.BoxDecoration{Color: col, Shape: painting.Circle}}
	}
	return w.App{Home: w.Column{
		CrossAxisAlignment: rendering.CrossAxisStretch,
		Children: []w.Widget{
			// Полоса как у AppBar: заголовок-заглушка слева.
			w.Container{
				Height: new(56.0), Color: gfx.Indigo, Padding: geom.Symmetric(16, 0), Alignment: &geom.CenterLeft,
				Child: w.Container{Width: new(120.0), Height: new(12.0),
					Decoration: &painting.BoxDecoration{Color: gfx.White.WithOpacity(0.8), BorderRadius: geom.RadiusAll(6)}},
			},
			// Две карточки 1:2.
			w.Padding{Padding: geom.All(16), Child: w.Row{Spacing: 12, Children: []w.Widget{
				w.Expanded{Child: w.Container{Height: new(80.0), Decoration: &painting.BoxDecoration{
					Color: gfx.White, BorderRadius: geom.RadiusAll(12),
					BoxShadow: []painting.BoxShadow{{Color: gfx.Black.WithOpacity(0.25), Offset: geom.Offset{Y: 3}, BlurRadius: 8}},
				}}},
				w.Expanded{Flex: 2, Child: w.Container{Height: new(80.0), Decoration: &painting.BoxDecoration{
					BorderRadius: geom.RadiusAll(12),
					Gradient:     painting.LinearGradient{Colors: []gfx.Color{gfx.Pink, gfx.Orange}},
				}}},
			}}},
			// Три круга с равными промежутками.
			w.Padding{Padding: geom.Symmetric(16, 0), Child: w.Row{
				MainAxisAlignment: rendering.MainAxisSpaceBetween,
				Children:          []w.Widget{circle(gfx.Red), circle(gfx.Green), circle(gfx.Blue)},
			}},
			// Остаток экрана: Stack с рамкой, кнопкой в углу и полупрозрачным квадратом по центру.
			w.Expanded{Child: w.Padding{Padding: geom.All(16), Child: w.Stack{Children: []w.Widget{
				w.Container{Decoration: &painting.BoxDecoration{Border: painting.BorderAll(gfx.Grey, 2), BorderRadius: geom.RadiusAll(16)}},
				w.Center{Child: w.Opacity{Opacity: 0.5, Child: w.Container{Width: new(80.0), Height: new(80.0), Color: gfx.Red}}},
				w.Positioned{Right: new(12.0), Bottom: new(12.0), Width: new(56.0), Height: new(56.0),
					Child: w.Container{Decoration: &painting.BoxDecoration{Color: gfx.Amber, Shape: painting.Circle,
						BoxShadow: []painting.BoxShadow{{Color: gfx.Black.WithOpacity(0.3), Offset: geom.Offset{Y: 4}, BlurRadius: 10}}}}},
			}}}},
			// Переносимые «чипы».
			w.Padding{Padding: geom.All(16), Child: w.Wrap{Spacing: 8, RunSpacing: 8, Children: []w.Widget{
				chip(70, gfx.Teal), chip(100, gfx.Purple), chip(60, gfx.Cyan), chip(90, gfx.Lime), chip(80, gfx.Brown),
			}}},
		},
	}}
}

func TestLayoutSceneGolden(t *testing.T) {
	hexagotest.Golden(t, "layout_scene", hexagotest.Render(layoutScene(), 600, 1000, 2))
}

func TestFlexDividesSpace(t *testing.T) {
	tree := w.Row{Children: []w.Widget{
		w.Expanded{Child: w.SizedBox{}},
		w.Expanded{Flex: 2, Child: w.SizedBox{}},
	}}
	binding := w.NewBinding(tree)
	binding.DrawFrame(gfx.NewImageCanvas(300, 100), w.MediaQueryData{Size: geom.Size{Width: 300, Height: 100}, DevicePixelRatio: 1})
	row := rendering.Child(binding.Root().FindBox()) // под корневым боксом экрана — наш Row
	children := rendering.Children(row)
	if len(children) != 2 {
		t.Fatalf("у ряда %d детей, ожидали 2", len(children))
	}
	if got := rendering.SizeOf(children[0]).Width; got != 100 {
		t.Errorf("первый ребёнок шириной %v, ожидали 100", got)
	}
	if got := rendering.SizeOf(children[1]).Width; got != 200 {
		t.Errorf("второй ребёнок шириной %v, ожидали 200", got)
	}
	if got := rendering.OffsetOf(children[1]).X; got != 100 {
		t.Errorf("второй ребёнок начинается с %v, ожидали 100", got)
	}
}

// tracked — виджет с состоянием, который запоминает свой порядковый номер.
type tracked struct {
	w.State
	Name string
	id   int
}

var created int

func (s *tracked) Init(w.BuildContext) {
	created++
	s.id = created
}

func (s *tracked) Build(w.BuildContext) w.Widget { return w.SizedBox{} }

func TestKeysPreserveStateOnReorder(t *testing.T) {
	created = 0
	list := func(names ...string) w.Widget {
		var children []w.Widget
		for _, n := range names {
			children = append(children, w.Keyed{Key: n, Child: tracked{Name: n}})
		}
		return w.Column{Children: children}
	}
	ids := func(b *w.Binding) map[string]int {
		out := map[string]int{}
		var walk func(e *w.Element)
		walk = func(e *w.Element) {
			if s, ok := e.State().(*tracked); ok {
				out[s.Name] = s.id
			}
			for _, c := range e.Children() {
				walk(c)
			}
		}
		walk(b.Root())
		return out
	}

	b := w.NewBinding(list("a", "b", "c"))
	frame := func() {
		b.DrawFrame(gfx.NewImageCanvas(10, 10), w.MediaQueryData{Size: geom.Size{Width: 10, Height: 10}, DevicePixelRatio: 1})
	}
	frame()
	before := ids(b)
	b.SetApp(list("c", "a", "b"))
	frame()
	after := ids(b)
	for _, n := range []string{"a", "b", "c"} {
		if before[n] != after[n] {
			t.Errorf("состояние %q пересоздано: было %d, стало %d", n, before[n], after[n])
		}
	}
	if created != 3 {
		t.Errorf("создано %d состояний, ожидали 3 — при перестановке ничего не должно создаваться", created)
	}
}

func TestBuildPanicShowsErrorBox(t *testing.T) {
	broken := w.Builder{Builder: func(w.BuildContext) w.Widget { panic("сломалось") }}
	img := hexagotest.Render(w.Center{Child: broken}, 200, 200, 1)
	if c := img.NRGBAAt(100, 100); c.R < 200 || c.G > 80 {
		t.Errorf("в центре ожидался красный ErrorBox, а там %v", c)
	}
}

func TestLayoutPanicPaintsRedFrame(t *testing.T) {
	// Растягивающийся на всю ширину бокс внутри Row — бесконечная ширина, ошибка лэйаута.
	tree := w.Row{Children: []w.Widget{w.SizedBoxExpand(nil)}}
	img := hexagotest.Render(tree, 100, 100, 1)
	if c := img.NRGBAAt(50, 50); c.R < 200 || c.G > 80 {
		t.Errorf("ожидался красный кадр с ошибкой, а в центре %v", c)
	}
}
