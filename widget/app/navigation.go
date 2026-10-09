package app

import (
	qt "github.com/mappu/miqt/qt6"
)

// PageFactory constructs one route page. Navigation passes the page host as
// parent; the host/stack then owns the returned widget.
type PageFactory func(parent *qt.QWidget) *qt.QWidget

type navigationEntry struct {
	route string
	page  *qt.QWidget
}

type RouteMap map[string]PageFactory

// Navigation owns route registration, stack semantics, and the Qt page host.
// It deliberately knows nothing about page business logic or notification.
type Navigation struct {
	Host *qt.QStackedWidget

	routes    RouteMap
	stack     []navigationEntry
	callbacks []routeCallback
}

type routeCallback struct {
	fn     func(string)
	active bool
}

func NewNavigation(host *qt.QStackedWidget) *Navigation {
	if host == nil {
		host = qt.NewQStackedWidget2()
	}
	host.SetContentsMargins(0, 0, 0, 0)
	return &Navigation{
		Host:      host,
		routes:    make(RouteMap),
		stack:     make([]navigationEntry, 0),
		callbacks: make([]routeCallback, 0),
	}
}

func (n *Navigation) register(route string, factory PageFactory) {
	if n == nil || route == "" || factory == nil {
		return
	}
	if n.routes == nil {
		n.routes = make(RouteMap)
	}
	n.routes[route] = factory
}

func (n *Navigation) RegisterAll(routes RouteMap) {
	for route, factory := range routes {
		n.register(route, factory)
	}
}

func (n *Navigation) Replace(route string) bool {
	if n == nil || n.Host == nil || route == "" {
		return false
	}
	if n.CurrentRoute() == route {
		return false
	}
	factory, ok := n.routes[route]
	if !ok || factory == nil {
		return false
	}
	page := factory(n.Host.QWidget)
	if page == nil {
		return false
	}
	setObjectName(page, route)

	if len(n.stack) == 0 {
		n.Host.AddWidget(page)
		n.stack = append(n.stack, navigationEntry{route: route, page: page})
	} else {
		old := n.stack[len(n.stack)-1]
		oldIndex := n.Host.IndexOf(old.page)
		if oldIndex >= 0 {
			n.Host.RemoveWidget(old.page)
		}
		old.page.Hide()
		old.page.DeleteLater()
		n.Host.AddWidget(page)
		n.stack[len(n.stack)-1] = navigationEntry{route: route, page: page}
	}
	n.Host.SetCurrentWidget(page)
	n.emitChanged(route)
	return true
}

func (n *Navigation) Push(route string) bool {
	if n == nil || n.Host == nil || route == "" {
		return false
	}
	factory, ok := n.routes[route]
	if !ok || factory == nil {
		return false
	}
	page := factory(n.Host.QWidget)
	if page == nil {
		return false
	}
	setObjectName(page, route)
	n.Host.AddWidget(page)
	n.stack = append(n.stack, navigationEntry{route: route, page: page})
	n.Host.SetCurrentWidget(page)
	n.emitChanged(route)
	return true
}

func (n *Navigation) Back() bool {
	if n == nil || n.Host == nil || len(n.stack) <= 1 {
		return false
	}
	current := n.stack[len(n.stack)-1]
	n.stack = n.stack[:len(n.stack)-1]
	index := n.Host.IndexOf(current.page)
	if index >= 0 {
		n.Host.RemoveWidget(current.page)
	}
	current.page.Hide()
	current.page.DeleteLater()
	previous := n.stack[len(n.stack)-1]
	n.Host.SetCurrentWidget(previous.page)
	n.emitChanged(previous.route)
	return true
}

func (n *Navigation) CurrentRoute() string {
	if n == nil || len(n.stack) == 0 {
		return ""
	}
	return n.stack[len(n.stack)-1].route
}

func (n *Navigation) Depth() int {
	if n == nil {
		return 0
	}
	return len(n.stack)
}

func (n *Navigation) CurrentPage() *qt.QWidget {
	if n == nil || n.Host == nil {
		return nil
	}
	return n.Host.CurrentWidget()
}

// OnRouteChanged registers a route observer and returns an unsubscribe
// function. Page-owned observers must unregister when their page is destroyed;
// otherwise a later route change retains the old page's closure.
func (n *Navigation) OnRouteChanged(fn func(string)) func() {
	if n == nil || fn == nil {
		return func() {}
	}
	index := len(n.callbacks)
	n.callbacks = append(n.callbacks, routeCallback{fn: fn, active: true})
	unsubscribed := false
	return func() {
		if unsubscribed {
			return
		}
		unsubscribed = true
		if index < len(n.callbacks) {
			n.callbacks[index].active = false
			n.callbacks[index].fn = nil
		}
	}
}

func (n *Navigation) emitChanged(route string) {
	for _, callback := range n.callbacks {
		if callback.active && callback.fn != nil {
			callback.fn(route)
		}
	}
}

func setObjectName(widget *qt.QWidget, name string) {
	if widget == nil || name == "" {
		return
	}
	view := qt.NewQAnyStringView3(name)
	defer view.Delete() // SetObjectName copies the view
	widget.SetObjectName(*view)
}
