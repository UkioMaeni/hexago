package material

import (
	"math"
	"time"

	"hexago/ui/animation"
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/rendering"
	"hexago/ui/text"
	"hexago/ui/widgets"
)

// TextField — поле ввода Material 3: залитое (по умолчанию) или с рамкой (Outlined). Подпись Label
// лежит в поле, пока оно пустое и не в фокусе, и поднимается вверх, когда в него вводят. Под полем —
// подсказка Helper или ошибка Error.
//
//	material.TextField{Label: "Почта", Keyboard: widgets.KeyboardEmail, Controller: c.email,
//		Leading: widgets.Icon{Icon: icons.Mail}, Error: c.emailError}
//
// Без Controller поле хранит текст само и сообщает о нём через OnChanged.
type TextField struct {
	widgets.State
	Controller *widgets.TextEditingController
	Focus      *widgets.FocusNode

	Outlined bool   // рамка вместо заливки
	Label    string // подпись в поле
	Hint     string // пример текста, пока поле пустое
	Helper   string // подсказка под полем
	Error    string // ошибка под полем (поле красное); заменяет Helper

	Leading  widgets.Widget // слева, обычно widgets.Icon
	Trailing widgets.Widget // справа, обычно widgets.Icon или IconButton (очистить, показать пароль)

	Multiline     bool
	Obscure       bool
	Keyboard      widgets.KeyboardType
	Action        widgets.TextInputAction
	NoAutocorrect bool
	Capitalize    widgets.Capitalization
	AutoFocus     bool
	ReadOnly      bool

	OnChanged   func(text string)
	OnSubmitted func(text string)

	ctrl      *widgets.TextEditingController
	focus     *widgets.FocusNode
	stopCtrl  func()
	stopFocus func()
}

func (f *TextField) Init(widgets.BuildContext) { f.sync() }

func (f *TextField) Update(widgets.BuildContext, TextField) { f.sync() }

func (f *TextField) Dispose() {
	f.stopCtrl()
	f.stopFocus()
}

// sync подписывается на контроллер и фокус (свои, если не заданы): от них зависят подпись и цвета.
func (f *TextField) sync() {
	if f.Controller != nil && f.Controller != f.ctrl || f.ctrl == nil {
		if f.stopCtrl != nil {
			f.stopCtrl()
		}
		f.ctrl = f.Controller
		if f.ctrl == nil {
			f.ctrl = &widgets.TextEditingController{}
		}
		f.stopCtrl = f.ctrl.AddListener(f.changed)
	}
	if f.Focus != nil && f.Focus != f.focus || f.focus == nil {
		if f.stopFocus != nil {
			f.stopFocus()
		}
		f.focus = f.Focus
		if f.focus == nil {
			f.focus = &widgets.FocusNode{}
		}
		f.stopFocus = f.focus.AddListener(f.changed)
	}
}

func (f *TextField) changed() {
	if f.Mounted() {
		f.SetState(nil)
	}
}

// Размеры Material 3 (dp).
const (
	fieldHeight  = 56.0
	fieldPadding = 16.0
	labelGap     = 4.0 // зазор в рамке вокруг поднятой подписи
)

func (f *TextField) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	focused := f.focus.HasFocus()
	empty := f.ctrl.Text() == ""
	errored := f.Error != ""

	accent := c.Primary
	if errored {
		accent = c.Error
	}
	input := t.Text.BodyLarge
	input.Color = c.OnSurface

	// Поля внутри: у залитого с подписью текст опущен под неё.
	pad := geom.EdgeInsets{Left: fieldPadding, Right: fieldPadding, Top: fieldPadding, Bottom: fieldPadding}
	if !f.Outlined && f.Label != "" {
		pad.Top, pad.Bottom = 24, 8
	}
	start := pad.Left // где начинается текст (и лежит подпись)
	if f.Leading != nil {
		pad.Left, start = 12, 12+24+16
	}
	if f.Trailing != nil {
		pad.Right = 12
	}

	var area widgets.Widget = widgets.EditableText{
		Controller: f.ctrl, Focus: f.focus, Style: input,
		CursorColor: accent, SelectionColor: c.Primary.WithOpacity(0.4),
		Multiline: f.Multiline, Obscure: f.Obscure, Keyboard: f.Keyboard, Action: f.Action,
		NoAutocorrect: f.NoAutocorrect, Capitalize: f.Capitalize, AutoFocus: f.AutoFocus, ReadOnly: f.ReadOnly,
		OnChanged: f.OnChanged, OnSubmitted: f.OnSubmitted,
		Controls:      TextSelectionControls{},
		ScrollPadding: geom.EdgeInsets{Left: 16, Top: 32, Right: 16, Bottom: 44}, // с подписью и подсказкой
	}
	// Место под подсказку есть всегда: если бы дерево меняло форму, поле ввода пересоздавалось бы
	// и теряло фокус.
	hint := t.Text.BodyLarge
	hint.Color = c.OnSurfaceVariant
	area = widgets.Stack{Children: []widgets.Widget{
		widgets.IgnorePointer{Child: widgets.Visibility{Visible: f.Hint != "" && empty && (f.Label == "" || focused),
			Child: widgets.Text{Data: f.Hint, Style: hint}}},
		area,
	}}
	row := []widgets.Widget{}
	iconColor := c.OnSurfaceVariant
	if f.Leading != nil {
		row = append(row, widgets.IconTheme{Data: widgets.IconThemeData{Size: 24, Color: iconColor}, Child: f.Leading},
			widgets.SizedBox{Width: new(16.0)})
	}
	// Отступы сверху и снизу — только у текста: значки стоят по центру поля, а кнопка-значок
	// не делает поле выше.
	row = append(row, widgets.Expanded{Child: widgets.Padding{Padding: geom.EdgeInsets{Top: pad.Top, Bottom: pad.Bottom}, Child: area}})
	if f.Trailing != nil {
		if errored {
			iconColor = c.Error
		}
		row = append(row, widgets.SizedBox{Width: new(16.0)},
			widgets.IconTheme{Data: widgets.IconThemeData{Size: 24, Color: iconColor}, Child: f.Trailing})
	}
	content := widgets.ConstrainedBox{
		Constraints: rendering.Constraints{MinHeight: fieldHeight, MaxWidth: rendering.Inf, MaxHeight: rendering.Inf},
		Child:       widgets.Padding{Padding: geom.EdgeInsets{Left: pad.Left, Right: pad.Right}, Child: widgets.Row{Children: row}},
	}

	floating := 0.0 // подпись поднята
	if focused || !empty {
		floating = 1
	}
	look := fieldLook{outlined: f.Outlined, focused: focused, errored: errored, label: f.Label, start: start,
		multiline: f.Multiline, theme: t}
	field := widgets.GestureDetector{
		OnTap: func() { f.focus.RequestFocus() }, // касание мимо текста (по полям рамки) тоже открывает ввод
		Child: widgets.Stack{Children: []widgets.Widget{
			widgets.PositionedFill(widgets.AnimatedValue{Value: floating, Duration: 150 * time.Millisecond,
				Curve: animation.Standard, Builder: look.background}),
			content,
			widgets.PositionedFill(widgets.IgnorePointer{Child: widgets.AnimatedValue{Value: floating,
				Duration: 150 * time.Millisecond, Curve: animation.Standard, Builder: look.labelAt}}),
		}},
	}

	support := f.Helper
	supportColor := c.OnSurfaceVariant
	if errored {
		support, supportColor = f.Error, c.Error
	}
	if support == "" {
		return field
	}
	st := t.Text.BodySmall
	st.Color = supportColor
	return widgets.Column{MainAxisSize: rendering.MainAxisMin, CrossAxisAlignment: rendering.CrossAxisStretch, Children: []widgets.Widget{
		field,
		widgets.Padding{Padding: geom.EdgeInsets{Left: 16, Right: 16, Top: 4}, Child: widgets.Text{Data: support, Style: st}},
	}}
}

// fieldLook — то, как выглядит поле при данной степени подъёма подписи.
type fieldLook struct {
	outlined, focused, errored, multiline bool
	label                                 string
	start                                 float64 // где лежит подпись в покое
	theme                                 ThemeData
}

// labelStyle — стиль подписи: в покое — BodyLarge, поднятая — BodySmall.
func (l fieldLook) labelStyle(k float64) text.Style {
	c := l.theme.Colors
	rest, up := l.theme.Text.BodyLarge, l.theme.Text.BodySmall
	col := c.OnSurfaceVariant
	switch {
	case l.errored:
		col = c.Error
	case l.focused:
		col = c.Primary
	}
	return text.Style{
		FontSize:      geom.Lerp(rest.FontSize, up.FontSize, k),
		LineHeight:    geom.Lerp(rest.LineHeight, up.LineHeight, k),
		LetterSpacing: geom.Lerp(rest.LetterSpacing, up.LetterSpacing, k),
		FontWeight:    rest.FontWeight,
		Color:         col,
	}
}

// labelPos — где левый верхний угол подписи.
func (l fieldLook) labelPos(k float64) geom.Offset {
	rest := geom.Offset{X: l.start, Y: (fieldHeight - 24) / 2}
	up := geom.Offset{X: l.start, Y: 8}
	if l.outlined {
		up = geom.Offset{X: fieldPadding, Y: -8} // посередине верхней линии рамки
	}
	return geom.Offset{X: geom.Lerp(rest.X, up.X, k), Y: geom.Lerp(rest.Y, up.Y, k)}
}

func (l fieldLook) labelAt(_ widgets.BuildContext, k float64) widgets.Widget {
	if l.label == "" {
		return widgets.SizedBox{}
	}
	p := l.labelPos(k)
	return widgets.Stack{Children: []widgets.Widget{widgets.Positioned{Left: new(p.X), Top: new(p.Y),
		Child: widgets.Text{Data: l.label, Style: l.labelStyle(k)}}}}
}

// background — заливка с линией снизу или рамка с зазором под поднятую подпись.
func (l fieldLook) background(_ widgets.BuildContext, k float64) widgets.Widget {
	c := l.theme.Colors
	line, width := c.OnSurfaceVariant, 1.0
	if l.outlined {
		line = c.Outline
	}
	switch {
	case l.errored:
		line = c.Error
	case l.focused:
		line = c.Primary
	}
	if l.focused {
		width = 2
	}
	radius := l.theme.Shapes.ExtraSmall
	gap := 0.0
	if l.outlined && l.label != "" && k > 0 {
		gap = k * (text.Layout(l.label, l.labelStyle(1), math.Inf(1)).Width + 2*labelGap)
	}
	return widgets.CustomPaint{Size: geom.Size{Width: widgets.Infinity, Height: widgets.Infinity},
		Painter: func(cv *gfx.Canvas, s geom.Size) {
			if !l.outlined {
				r := geom.Circular(radius)
				cv.DrawRRect(geom.RRect{Rect: geom.RectFromSize(geom.Zero, s), Radius: geom.BorderRadius{TopLeft: r, TopRight: r}},
					gfx.Paint{Color: c.SurfaceContainerHighest})
				cv.DrawRect(geom.Rect{Top: s.Height - width, Right: s.Width, Bottom: s.Height}, gfx.Paint{Color: line})
				return
			}
			rr := geom.RRect{Rect: geom.RectFromSize(geom.Zero, s).Inflate(-width / 2), Radius: geom.RadiusAll(radius)}
			p := gfx.Paint{Color: line, Style: gfx.Stroke, StrokeWidth: width}
			if gap <= 0 {
				cv.DrawRRect(rr, p)
				return
			}
			// Рамка с зазором: три куска — слева от зазора, справа от него и всё ниже верхней линии.
			x0 := fieldPadding - labelGap
			for _, clip := range []geom.Rect{
				{Right: x0, Bottom: s.Height},
				{Left: x0 + gap, Right: s.Width, Bottom: s.Height},
				{Left: x0, Top: width + 1, Right: x0 + gap, Bottom: s.Height},
			} {
				cv.Save()
				cv.ClipRect(clip)
				cv.DrawRRect(rr, p)
				cv.Restore()
			}
		}}
}
