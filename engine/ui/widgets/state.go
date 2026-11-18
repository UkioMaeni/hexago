package widgets

import (
	"fmt"
	"reflect"
)

// State встраивается вместо Base в виджет, у которого есть своё изменяемое состояние:
//
//	type Counter struct {
//		widgets.State
//
//		Title string // с большой буквы — параметр: его задаёт родитель
//		count int    // с маленькой — своё состояние: живёт между перестройками
//	}
//
//	func (c *Counter) Init(ctx widgets.BuildContext) { c.count = 10 }
//
//	func (c *Counter) Build(ctx widgets.BuildContext) widgets.Widget {
//		return widgets.GestureDetector{
//			OnTap: func() { c.SetState(func() { c.count++ }) },
//			Child: widgets.Text{Data: fmt.Sprint(c.Title, ": ", c.count)},
//		}
//	}
//
// Родитель пишет виджет как обычно: Counter{Title: "Нажатия"}. Когда виджет впервые появляется
// в дереве, Hexago создаёт его живой экземпляр и хранит его, пока виджет в дереве. Родитель
// перестраивается — в живой экземпляр переписываются только параметры (поля с большой буквы),
// а состояние (поля с маленькой) остаётся.
//
// Методы такого виджета объявляются у указателя (func (c *Counter) ...), иначе они меняли бы копию.
// Кроме Build, по желанию:
//
//	Init(ctx BuildContext)                — виджет появился в дереве (один раз, до первого Build);
//	Update(ctx BuildContext, old Counter) — родитель перестроился и передал параметры; old — прежние;
//	DependenciesChanged(ctx BuildContext) — изменились данные предков, полученные через DependOn;
//	Dispose()                             — виджет убран из дерева: остановить таймеры, отписаться.
type State struct {
	Base
	element *Element
}

func (s *State) state() *State { return s }

// SetState меняет состояние и перестраивает виджет в следующем кадре, как setState во Flutter:
//
//	c.SetState(func() { c.count++ })
func (s *State) SetState(fn func()) {
	if s.element == nil || !s.element.mounted {
		panic("widgets: SetState у виджета, которого нет в дереве: до Init или после Dispose")
	}
	if fn != nil {
		fn()
	}
	s.element.markNeedsBuild()
}

// Context — место виджета в дереве (то же, что ctx в Build).
func (s *State) Context() BuildContext { return s.element }

// Mounted сообщает, что виджет в дереве: после Dispose — false.
func (s *State) Mounted() bool { return s.element != nil && s.element.mounted }

// Методы живого экземпляра виджета с состоянием.
type (
	stateHolder interface {
		Widget
		state() *State
	}
	builder             interface{ Build(ctx BuildContext) Widget }
	initer              interface{ Init(ctx BuildContext) }
	disposer            interface{ Dispose() }
	dependenciesChanger interface{ DependenciesChanged(ctx BuildContext) }
)

var (
	stateHolderType = reflect.TypeFor[stateHolder]()
	stateType       = reflect.TypeFor[State]()
	contextType     = reflect.TypeFor[BuildContext]()
)

// lifecycle — методы, которые у виджета с состоянием должны быть объявлены у указателя.
var lifecycle = []string{"Build", "Init", "Update", "DependenciesChanged", "Dispose"}

// isStateful сообщает, что тип — структура со встроенным State.
func isStateful(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && reflect.PointerTo(t).Implements(stateHolderType)
}

// mountState создаёт живой экземпляр виджета, связывает его с элементом и вызывает Init.
func (e *Element) mountState() {
	t := reflect.TypeOf(e.widget)
	for _, name := range lifecycle {
		if _, ok := t.MethodByName(name); ok {
			panic(fmt.Sprintf("widgets: у виджета с состоянием методы объявляются у указателя: "+
				"func (x *%s) %s(...), а не func (x %s) — иначе изменения состояния теряются", t.Name(), name, t.Name()))
		}
	}
	live := reflect.New(t)
	live.Elem().Set(reflect.ValueOf(e.widget))
	if _, ok := live.Interface().(builder); !ok {
		panic(fmt.Sprintf("widgets: у %s нет метода Build: func (x *%s) Build(ctx widgets.BuildContext) widgets.Widget",
			t.Name(), t.Name()))
	}
	checkUpdate(t, live)
	live.Interface().(stateHolder).state().element = e
	e.live = live
	if s, ok := live.Interface().(initer); ok {
		s.Init(e)
	}
}

// updateState переписывает в живой экземпляр параметры нового виджета и вызывает Update.
func (e *Element) updateState(w Widget) {
	cur := e.live.Elem()
	old := reflect.New(cur.Type()).Elem()
	old.Set(cur)
	copyParams(cur, reflect.ValueOf(w))
	if m := e.live.MethodByName("Update"); m.IsValid() {
		m.Call([]reflect.Value{reflect.ValueOf(e), old})
	}
}

// copyParams переписывает экспортируемые поля (параметры) из src в dst;
// неэкспортируемые (состояние) и сам State остаются как были.
func copyParams(dst, src reflect.Value) {
	t := dst.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() || f.Type == stateType {
			continue
		}
		dst.Field(i).Set(src.Field(i))
	}
}

// checkUpdate проверяет, что Update объявлен правильно: func (x *T) Update(ctx BuildContext, old T).
// Ошибка в сигнатуре иначе молча означала бы, что метод не вызывается.
func checkUpdate(t reflect.Type, live reflect.Value) {
	m := live.MethodByName("Update")
	if !m.IsValid() {
		return
	}
	mt := m.Type()
	if mt.NumIn() != 2 || mt.In(0) != contextType || mt.In(1) != t || mt.NumOut() != 0 {
		panic(fmt.Sprintf("widgets: метод Update у %s должен быть таким: "+
			"func (x *%s) Update(ctx widgets.BuildContext, old %s)", t.Name(), t.Name(), t.Name()))
	}
}
