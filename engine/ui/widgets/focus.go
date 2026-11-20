package widgets

import (
	"fmt"
	"runtime/debug"
	"slices"
)

// FocusNode — фокус ввода у виджета (поля ввода): в фокусе ровно одно поле, к нему подключена
// клавиатура. Обычно поле создаёт узел само; свой узел нужен, чтобы управлять фокусом из кода:
//
//	c.email = &widgets.FocusNode{}
//	material.TextField{Focus: c.email}
//	...
//	c.email.RequestFocus() // клавиатура откроется для этого поля
type FocusNode struct {
	owner     *BuildOwner
	element   *Element // виджет, которому принадлежит узел (для порядка «Далее»)
	focused   bool
	want      bool // RequestFocus раньше, чем виджет появился в дереве
	listeners []*func()
}

// HasFocus — виджет сейчас в фокусе.
func (f *FocusNode) HasFocus() bool { return f.focused }

// RequestFocus переводит фокус на этот виджет (и подключает к нему клавиатуру).
func (f *FocusNode) RequestFocus() {
	if f.owner == nil {
		f.want = true
		return
	}
	f.owner.setFocus(f)
}

// Unfocus снимает фокус с виджета (клавиатура прячется, если фокус никуда не перешёл).
func (f *FocusNode) Unfocus() {
	f.want = false
	if f.owner != nil && f.owner.focus == f {
		f.owner.setFocus(nil)
	}
}

// AddListener — fn вызывается, когда виджет получает или теряет фокус. Возвращает отписку.
func (f *FocusNode) AddListener(fn func()) (remove func()) {
	p := &fn
	f.listeners = append(f.listeners, p)
	return func() {
		if i := slices.Index(f.listeners, p); i >= 0 {
			f.listeners = slices.Delete(f.listeners, i, i+1)
		}
	}
}

func (f *FocusNode) notify() {
	for _, l := range slices.Clone(f.listeners) {
		(*l)()
	}
}

// attach связывает узел с виджетом в дереве.
func (f *FocusNode) attach(e *Element) {
	f.owner, f.element = e.owner, e
	e.owner.focusNodes = append(e.owner.focusNodes, f)
	if f.want {
		f.want = false
		f.owner.setFocus(f)
	}
}

// detach — виджет ушёл из дерева: фокус с него снимается.
func (f *FocusNode) detach() {
	o := f.owner
	if o == nil {
		return
	}
	if i := slices.Index(o.focusNodes, f); i >= 0 {
		o.focusNodes = slices.Delete(o.focusNodes, i, i+1)
	}
	if o.focus == f {
		o.setFocus(nil)
	}
	f.owner, f.element = nil, nil
}

// ClearFocus снимает фокус с любого поля — клавиатура прячется.
func ClearFocus(ctx BuildContext) { ctx.element().owner.setFocus(nil) }

// setFocus переводит фокус на n (nil — никуда). Клавиатура прячется, только если после этого
// ни одно поле к ней не подключилось: при переходе от поля к полю она не мигает.
func (o *BuildOwner) setFocus(n *FocusNode) {
	if o.focus == n {
		return
	}
	old := o.focus
	o.focus = n
	if old != nil {
		old.focused = false
		old.notify()
	}
	if n != nil {
		n.focused = true
		n.notify()
	}
	if o.textClient == nil && o.keyboardShown && o.textInput != nil {
		o.keyboardShown = false
		o.textInput.Hide()
	}
}

// nextFocus — следующий после n узел фокуса в порядке дерева, в пределах того же экрана
// (маршрута Navigator); nil — n последний.
func (o *BuildOwner) nextFocus(n *FocusNode) *FocusNode {
	if n.element == nil {
		return nil
	}
	scope := n.element
	for p := n.element.parent; p != nil; p = p.parent {
		scope = p
		if _, ok := p.widget.(routeScope); ok {
			break
		}
	}
	byElement := map[*Element]*FocusNode{}
	for _, f := range o.focusNodes {
		if f.element != nil {
			byElement[f.element] = f
		}
	}
	var order []*FocusNode
	var walk func(e *Element)
	walk = func(e *Element) {
		if f := byElement[e]; f != nil {
			order = append(order, f)
		}
		for _, c := range e.children {
			walk(c)
		}
	}
	walk(scope)
	if i := slices.Index(order, n); i >= 0 && i+1 < len(order) {
		return order[i+1]
	}
	return nil
}

// recoverEvent — ошибка в обработчике события не роняет приложение (как ошибка в Build).
func (b *Binding) recoverEvent(what string) {
	if r := recover(); r != nil {
		fmt.Printf("Ошибка при обработке %s: %v\n%s\n", what, r, debug.Stack())
	}
}
