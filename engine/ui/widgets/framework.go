// Package widgets — декларативный интерфейс Hexago, как widgets-библиотека Flutter.
//
// Три дерева (ADR-0002):
//   - Widget — неизменяемое описание, создаётся заново при каждом построении;
//   - Element — экземпляр виджета в дереве: живёт долго, хранит состояние виджета, решает,
//     обновить существующее поддерево или пересоздать его;
//   - rendering.Box — геометрия: лэйаут, отрисовка, касания.
//
// Вид виджета определяется методами, которые он реализует:
//   - Build(ctx) Widget — виджет строится из других виджетов;
//   - встроен State вместо Base — у виджета своё изменяемое состояние (см. State);
//   - CreateRenderObject / UpdateRenderObject — виджет с собственным render-боксом.
package widgets

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"slices"

	"hexago/ui/animation"
	"hexago/ui/gestures"
	"hexago/ui/rendering"
)

// Widget — описание части интерфейса. Виджетом становится любой тип, встроивший Base.
type Widget interface {
	isWidget()
}

// Base встраивается в каждый виджет: `type MyWidget struct { widgets.Base; ... }`.
type Base struct{}

func (Base) isWidget() {}

// BuildContext — место виджета в дереве. Через него виджет узнаёт данные предков
// (тему, размер экрана) функцией DependOn.
type BuildContext interface {
	// Widget — виджет, которому принадлежит контекст.
	Widget() Widget
	element() *Element
}

// StatelessWidget — виджет без своего состояния: он целиком описывается полями.
type StatelessWidget interface {
	Widget
	Build(ctx BuildContext) Widget
}

// RenderObjectWidget — виджет со своим render-боксом. Детей он отдаёт через
// ChildWidget() (один ребёнок) или ChildWidgets() (несколько).
type RenderObjectWidget interface {
	Widget
	CreateRenderObject(ctx BuildContext) rendering.Box
	UpdateRenderObject(ctx BuildContext, b rendering.Box)
}

// SingleChildWidget — виджет с одним ребёнком.
type SingleChildWidget interface {
	ChildWidget() Widget
}

// MultiChildWidget — виджет с несколькими детьми.
type MultiChildWidget interface {
	ChildWidgets() []Widget
}

// InheritedWidget — данные, видимые всему поддереву (тема, размер экрана).
// Потомки получают их через DependOn и перестраиваются, когда данные меняются.
type InheritedWidget interface {
	Widget
	SingleChildWidget
	// UpdateShouldNotify сообщает, изменились ли данные по сравнению со старым виджетом.
	UpdateShouldNotify(old Widget) bool
}

// ParentDataWidget — обёртка, которая сообщает родительскому боксу данные о ребёнке
// (Expanded для Row/Column, Positioned для Stack).
type ParentDataWidget interface {
	Widget
	SingleChildWidget
	ApplyParentData(b rendering.Box)
}

// Keyed даёт ребёнку ключ: при перестройке списка элемент с тем же ключом сохраняет
// своё состояние, даже если переехал на другое место. Ключ — любое сравнимое значение.
type Keyed struct {
	Base
	Key   any
	Child Widget
}

// ChildWidget — ребёнок.
func (k Keyed) ChildWidget() Widget { return k.Child }

// DependOn находит ближайшего предка-InheritedWidget типа T и подписывает виджет
// на его изменения, как dependOnInheritedWidgetOfExactType во Flutter.
func DependOn[T InheritedWidget](ctx BuildContext) (T, bool) {
	var zero T
	e := ctx.element()
	ie := e.inherited[reflect.TypeFor[T]()]
	if ie == nil {
		return zero, false
	}
	ie.dependents[e] = true
	e.deps[ie] = true
	return ie.widget.(T), true
}

type kind uint8

const (
	kindStateless kind = iota
	kindStateful
	kindRender
	kindInherited
	kindParentData
	kindProxy // Keyed и другие обёртки с одним ребёнком
)

func kindOf(w Widget) kind {
	t := reflect.TypeOf(w)
	if t.Kind() == reflect.Pointer && isStateful(t.Elem()) {
		panic(fmt.Sprintf("widgets: виджет с состоянием передают значением: %s{...}, а не &%s{...}",
			t.Elem().Name(), t.Elem().Name()))
	}
	if isStateful(t) {
		return kindStateful
	}
	switch w.(type) {
	case RenderObjectWidget:
		return kindRender
	case StatelessWidget:
		return kindStateless
	case InheritedWidget:
		return kindInherited
	case ParentDataWidget:
		return kindParentData
	case SingleChildWidget:
		return kindProxy
	}
	panic(fmt.Sprintf("widgets: тип %T встроил widgets.Base, но у него нет ни Build, "+
		"ни CreateRenderObject — непонятно, как его строить", w))
}

// Element — экземпляр виджета в дереве.
type Element struct {
	widget     Widget
	parent     *Element
	children   []*Element
	depth      int
	owner      *BuildOwner
	kind       kind
	live       reflect.Value           // у виджета с состоянием — живой экземпляр (*ТвойВиджет)
	animations []*animation.Controller // созданы через NewAnimation; останавливаются при удалении
	box        rendering.Box
	dirty      bool
	mounted    bool

	inherited  map[reflect.Type]*Element // видимые отсюда InheritedWidget по типу
	dependents map[*Element]bool         // у InheritedWidget: кто от него зависит
	deps       map[*Element]bool         // от каких InheritedWidget зависит этот элемент
}

// Widget — текущий виджет элемента.
func (e *Element) Widget() Widget { return e.widget }

func (e *Element) element() *Element { return e }

// State — живой экземпляр виджета с состоянием (*ТвойВиджет); nil у остальных виджетов.
func (e *Element) State() Widget {
	if !e.live.IsValid() {
		return nil
	}
	return e.live.Interface().(Widget)
}

// Children — дочерние элементы.
func (e *Element) Children() []*Element { return e.children }

// Box — render-бокс элемента (nil, если у виджета нет своего бокса).
func (e *Element) Box() rendering.Box { return e.box }

// FindBox — первый render-бокс в поддереве элемента (для Stateless/Stateful это бокс их потомка).
func (e *Element) FindBox() rendering.Box {
	if e.box != nil {
		return e.box
	}
	if boxes := e.renderDescendants(nil); len(boxes) > 0 {
		return boxes[0]
	}
	return nil
}

// BuildOwner управляет перестройками: помнит «грязные» элементы и перестраивает их по порядку.
type BuildOwner struct {
	dirty     []*Element
	gestures  gestures.Env        // арена и часы для распознавателей жестов (их задаёт Binding)
	scheduler animation.Scheduler // кадры для анимаций (Binding)
	// OnNeedsBuild вызывается, когда появился первый «грязный» элемент — пора просить кадр.
	OnNeedsBuild func()
}

func (e *Element) markNeedsBuild() {
	if !e.mounted || e.dirty {
		return
	}
	e.dirty = true
	o := e.owner
	o.dirty = append(o.dirty, e)
	if len(o.dirty) == 1 && o.OnNeedsBuild != nil {
		o.OnNeedsBuild()
	}
}

// BuildScope перестраивает все «грязные» элементы: сначала верхние (по глубине),
// чтобы нижние не строились дважды.
func (o *BuildOwner) BuildScope() {
	for len(o.dirty) > 0 {
		batch := o.dirty
		o.dirty = nil
		slices.SortStableFunc(batch, func(a, b *Element) int { return a.depth - b.depth })
		for _, e := range batch {
			if e.mounted && e.dirty {
				e.rebuild()
			}
		}
	}
}

// Mount создаёт корневой элемент дерева.
func (o *BuildOwner) Mount(w Widget) *Element { return o.mount(nil, w) }

// Update обновляет корень дерева новым виджетом.
func (o *BuildOwner) Update(root *Element, w Widget) {
	if !canUpdate(root.widget, w) {
		panic("widgets: у корня дерева нельзя менять тип виджета")
	}
	root.update(w)
}

func (o *BuildOwner) mount(parent *Element, w Widget) *Element {
	e := &Element{widget: w, parent: parent, owner: o, mounted: true, kind: kindOf(w), deps: map[*Element]bool{}}
	if parent != nil {
		e.depth = parent.depth + 1
		e.inherited = parent.inherited
	}
	switch e.kind {
	case kindInherited:
		m := make(map[reflect.Type]*Element, len(e.inherited)+1)
		for k, v := range e.inherited {
			m[k] = v
		}
		m[reflect.TypeOf(w)] = e
		e.inherited = m
		e.dependents = map[*Element]bool{}
	case kindStateful:
		e.mountState()
	case kindRender:
		e.box = w.(RenderObjectWidget).CreateRenderObject(e)
		rendering.MarkNeedsPaint(e.box)
	}
	e.rebuild()
	return e
}

func (o *BuildOwner) unmount(e *Element) {
	for _, c := range e.children {
		o.unmount(c)
	}
	if e.kind == kindStateful {
		if s, ok := e.live.Interface().(disposer); ok {
			s.Dispose()
		}
	}
	for _, a := range e.animations {
		a.Dispose()
	}
	e.animations = nil
	for dep := range e.deps {
		delete(dep.dependents, e)
	}
	e.mounted = false
	e.children = nil
}

// rebuild строит детей элемента заново и сверяет их со старыми.
func (e *Element) rebuild() {
	e.dirty = false
	switch e.kind {
	case kindStateless:
		e.updateSingle(e.safeBuild(func() Widget { return e.widget.(StatelessWidget).Build(e) }))
	case kindStateful:
		e.updateSingle(e.safeBuild(func() Widget { return e.live.Interface().(builder).Build(e) }))
	case kindRender:
		switch w := e.widget.(type) {
		case MultiChildWidget:
			e.children = e.owner.updateChildren(e, e.children, w.ChildWidgets())
		case SingleChildWidget:
			e.updateSingle(w.ChildWidget())
		}
	default:
		e.updateSingle(e.widget.(SingleChildWidget).ChildWidget())
	}
}

// safeBuild вызывает построение; если оно паникует, на месте виджета появляется ErrorBox,
// а текст ошибки печатается — как красный экран ошибки во Flutter.
func (e *Element) safeBuild(build func() Widget) (w Widget) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Ошибка при построении %T: %v\n%s\n", e.widget, r, debug.Stack())
			w = ErrorBox{Message: fmt.Sprint(r)}
		}
	}()
	return build()
}

func (e *Element) updateSingle(w Widget) {
	var old *Element
	if len(e.children) > 0 {
		old = e.children[0]
	}
	if n := e.owner.updateChild(e, old, w); n != nil {
		e.children = []*Element{n}
	} else {
		e.children = nil
	}
}

// update применяет к элементу новый виджет того же типа и ключа.
func (e *Element) update(w Widget) {
	old := e.widget
	e.widget = w
	switch e.kind {
	case kindStateful:
		e.updateState(w)
	case kindRender:
		snap := rendering.Snapshot(e.box)
		w.(RenderObjectWidget).UpdateRenderObject(e, e.box)
		rendering.MarkIfChanged(e.box, snap)
	case kindInherited:
		if w.(InheritedWidget).UpdateShouldNotify(old) {
			for dep := range e.dependents {
				switch dep.kind {
				case kindStateful:
					if s, ok := dep.live.Interface().(dependenciesChanger); ok {
						s.DependenciesChanged(dep)
					}
				case kindRender:
					// Бокс берёт данные предка при создании (Text — стиль по умолчанию): обновить его.
					snap := rendering.Snapshot(dep.box)
					dep.widget.(RenderObjectWidget).UpdateRenderObject(dep, dep.box)
					rendering.MarkIfChanged(dep.box, snap)
				}
				dep.markNeedsBuild()
			}
		}
	}
	e.rebuild()
}

// updateChild — сердце reconciliation: старый элемент обновляется, если новый виджет
// того же типа и с тем же ключом, иначе заменяется новым.
func (o *BuildOwner) updateChild(parent, old *Element, w Widget) *Element {
	if w == nil {
		if old != nil {
			o.unmount(old)
		}
		return nil
	}
	if old != nil {
		if canUpdate(old.widget, w) {
			old.update(w)
			return old
		}
		o.unmount(old)
	}
	return o.mount(parent, w)
}

// updateChildren сверяет список детей: элементы с ключами находятся по ключу,
// остальные — по порядку.
func (o *BuildOwner) updateChildren(parent *Element, old []*Element, ws []Widget) []*Element {
	keyed := map[any]*Element{}
	var unkeyed []*Element
	for _, e := range old {
		if k := keyOf(e.widget); k != nil {
			keyed[k] = e
		} else {
			unkeyed = append(unkeyed, e)
		}
	}
	out := make([]*Element, 0, len(ws))
	for _, w := range ws {
		if w == nil {
			continue
		}
		if k := keyOf(w); k != nil {
			if e, ok := keyed[k]; ok && canUpdate(e.widget, w) {
				delete(keyed, k)
				e.update(w)
				out = append(out, e)
			} else {
				out = append(out, o.mount(parent, w))
			}
			continue
		}
		if len(unkeyed) > 0 {
			e := unkeyed[0]
			unkeyed = unkeyed[1:]
			if canUpdate(e.widget, w) {
				e.update(w)
				out = append(out, e)
				continue
			}
			o.unmount(e)
		}
		out = append(out, o.mount(parent, w))
	}
	for _, e := range keyed {
		o.unmount(e)
	}
	for _, e := range unkeyed {
		o.unmount(e)
	}
	return out
}

func canUpdate(a, b Widget) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b) && keyOf(a) == keyOf(b)
}

func keyOf(w Widget) any {
	if k, ok := w.(Keyed); ok {
		return k.Key
	}
	return nil
}

// renderDescendants — ближайшие render-боксы под элементом (сам элемент не входит).
func (e *Element) renderDescendants(out []rendering.Box) []rendering.Box {
	for _, c := range e.children {
		if c.box != nil {
			out = append(out, c.box)
		} else {
			out = c.renderDescendants(out)
		}
	}
	return out
}

// SyncRenderTree связывает render-боксы так же, как связаны элементы:
// детьми бокса становятся ближайшие боксы под его элементом.
// Заодно ParentDataWidget (Expanded, Positioned) передают данные своим боксам.
func SyncRenderTree(e *Element) { syncRenderTree(e, nil) }

// syncRenderTree — то же; removed получает боксы, которые ушли из дерева (их место надо перерисовать).
func syncRenderTree(e *Element, removed func(rendering.Box)) {
	if e.box != nil {
		next := e.renderDescendants(nil)
		if removed != nil {
			for _, old := range rendering.Children(e.box) {
				if !slices.Contains(next, old) {
					removed(old)
				}
			}
		}
		rendering.SetChildren(e.box, next)
	}
	if e.kind == kindParentData {
		if b := e.FindBox(); b != nil {
			e.widget.(ParentDataWidget).ApplyParentData(b)
		}
	}
	for _, c := range e.children {
		syncRenderTree(c, removed)
	}
}
