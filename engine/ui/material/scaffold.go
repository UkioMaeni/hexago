package material

import (
	"hexago/ui/geom"
	"hexago/ui/gfx"
	"hexago/ui/icons"
	"hexago/ui/rendering"
	"hexago/ui/widgets"
)

// Scaffold — каркас экрана Material: AppBar сверху, Body, плавающая кнопка в правом нижнем углу,
// нижняя панель (NavigationBar) и боковая панель (Drawer). Фон — Surface темы. Внизу, над нижней
// панелью, появляются сообщения SnackBar — плавающая кнопка поднимается над ними.
//
// Отступы системных панелей Scaffold отдаёт тем, кто у края: AppBar сам отступает от строки
// состояния, нижняя панель — от панели навигации. Для Body эти отступы убираются из MediaQuery,
// поэтому SafeArea внутри Body не отступит второй раз.
type Scaffold struct {
	widgets.Base
	AppBar               widgets.Widget
	Body                 widgets.Widget
	FloatingActionButton widgets.Widget
	BottomBar            widgets.Widget
	// Drawer — боковая панель (обычно NavigationDrawer): AppBar ставит кнопку «меню», панель
	// выдвигается и движением пальца от левого края экрана.
	Drawer          widgets.Widget
	BackgroundColor gfx.Color // прозрачный — Surface
}

func (s Scaffold) Build(ctx widgets.BuildContext) widgets.Widget {
	bg := s.BackgroundColor
	if bg.IsTransparent() {
		bg = Of(ctx).Colors.Surface
	}
	mq := widgets.MediaQueryOf(ctx)
	bodyMQ := mq
	if s.AppBar != nil {
		bodyMQ.Padding.Top = 0
	}
	if s.BottomBar != nil {
		bodyMQ.Padding.Bottom = 0
	}
	bottom := 0.0
	if s.BottomBar == nil {
		bottom = mq.Padding.Bottom
	}
	layers := []widgets.Widget{
		widgets.PositionedFill(widgets.MediaQuery{Data: bodyMQ, Child: s.Body}),
		widgets.Positioned{Left: new(0.0), Right: new(0.0), Bottom: new(bottom), Child: scaffoldBottom{
			FAB: s.FloatingActionButton, Left: mq.Padding.Left, Right: mq.Padding.Right,
		}},
	}
	column := []widgets.Widget{}
	if s.AppBar != nil {
		column = append(column, s.AppBar)
	}
	column = append(column, widgets.Expanded{Child: widgets.Stack{Children: layers}})
	if s.BottomBar != nil {
		column = append(column, s.BottomBar)
	}
	var root widgets.Widget = widgets.ColoredBox{Color: bg,
		Child: widgets.Column{CrossAxisAlignment: rendering.CrossAxisStretch, Children: column}}
	if s.Drawer != nil {
		root = widgets.Stack{Fit: rendering.StackExpand, Children: []widgets.Widget{root,
			widgets.Positioned{Left: new(0.0), Top: new(0.0), Bottom: new(0.0),
				Child: drawerEdge{Drawer: s.Drawer, Width: 20 + mq.Padding.Left}},
		}}
	}
	return scaffoldScope{drawer: s.Drawer, Child: root}
}

// AppBar — верхняя панель Material 3 (small top app bar): высота 64 dp под строкой состояния,
// Leading слева (обычно IconButton «меню»), заголовок, Actions справа. Если Leading не задан, слева сама
// появляется кнопка «меню» (у Scaffold есть Drawer) или стрелка «Назад» (с экрана есть куда вернуться).
type AppBar struct {
	widgets.Base
	Leading      widgets.Widget
	NoBackButton bool           // не ставить стрелку «Назад» самой
	Title        widgets.Widget // обычно widgets.Text без стиля
	Actions      []widgets.Widget
	CenterTitle  bool
	Color        gfx.Color // прозрачный — Surface
}

func (a AppBar) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	bg := a.Color
	if bg.IsTransparent() {
		bg = t.Colors.Surface
	}
	title := t.Text.TitleLarge
	title.Color = t.Colors.OnSurface

	leading := a.Leading
	if s, ok := widgets.DependOn[scaffoldScope](ctx); leading == nil && ok && s.drawer != nil {
		leading = IconButton{Icon: icons.Menu, OnPressed: func() { OpenDrawer(ctx) }}
	}
	if leading == nil && !a.NoBackButton && widgets.CanPop(ctx) {
		leading = IconButton{Icon: icons.ArrowBack, OnPressed: func() { widgets.Pop(ctx) }}
	}
	row := []widgets.Widget{}
	start := 16.0
	if leading != nil {
		row = append(row, widgets.Padding{Padding: geom.EdgeInsets{Left: 4}, Child: leading})
		start = 4
	}
	var center widgets.Widget = widgets.SizedBox{}
	if a.Title != nil {
		center = widgets.DefaultTextStyle{Style: title, Child: a.Title}
		if a.CenterTitle {
			center = widgets.Center{Child: center}
		}
	}
	row = append(row, widgets.Expanded{Child: widgets.Padding{Padding: geom.EdgeInsets{Left: start, Right: 16}, Child: center}})
	if len(a.Actions) > 0 {
		row = append(row, widgets.Padding{Padding: geom.EdgeInsets{Right: 4},
			Child: widgets.Row{MainAxisSize: rendering.MainAxisMin, Children: a.Actions}})
	}
	top := widgets.MediaQueryOf(ctx).Padding.Top
	return Material{Color: bg, Child: widgets.Padding{Padding: geom.EdgeInsets{Top: top}, Child: widgets.SizedBox{
		Height: new(64.0),
		Child: widgets.IconTheme{Data: widgets.IconThemeData{Size: 24, Color: t.Colors.OnSurface},
			Child: widgets.Row{Children: row}},
	}}}
}

// NavigationDestination — пункт нижней навигации.
type NavigationDestination struct {
	Icon         *icons.Data
	SelectedIcon *icons.Data // значок выбранного пункта; nil — тот же Icon
	Label        string
}

// NavigationBar — нижняя навигация Material 3: 3–5 пунктов, высота 80 dp над панелью навигации
// системы. У выбранного пункта значок на «пилюле» SecondaryContainer.
type NavigationBar struct {
	widgets.Base
	Destinations []NavigationDestination
	Selected     int
	OnSelected   func(int)
}

func (n NavigationBar) Build(ctx widgets.BuildContext) widgets.Widget {
	t := Of(ctx)
	c := t.Colors
	items := make([]widgets.Widget, 0, len(n.Destinations))
	for i, d := range n.Destinations {
		selected := i == n.Selected
		icon, iconColor, indicator := d.Icon, c.OnSurfaceVariant, gfx.Transparent
		label := t.Text.LabelMedium
		label.Color = c.OnSurfaceVariant
		if selected {
			iconColor, indicator, label.Color = c.OnSecondaryContainer, c.SecondaryContainer, c.OnSurface
			if d.SelectedIcon != nil {
				icon = d.SelectedIcon
			}
		}
		var tap func()
		if n.OnSelected != nil {
			tap = func() { n.OnSelected(i) }
		}
		items = append(items, widgets.Expanded{Child: InkWell{OnTap: tap, Child: widgets.Column{
			MainAxisAlignment: rendering.MainAxisCenter, Spacing: 4,
			Children: []widgets.Widget{
				Material{Color: indicator, Radius: 16, Child: widgets.SizedBox{Width: new(64.0), Height: new(32.0),
					Child: widgets.Center{Child: widgets.Icon{Icon: icon, Size: 24, Color: iconColor}}}},
				widgets.Text{Data: d.Label, Style: label},
			},
		}}})
	}
	bottom := widgets.MediaQueryOf(ctx).Padding.Bottom
	return Material{Color: c.SurfaceContainer, Child: widgets.Padding{Padding: geom.EdgeInsets{Bottom: bottom},
		Child: widgets.SizedBox{Height: new(80.0), Child: widgets.Row{Children: items}}}}
}
