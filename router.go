package easytcp

import (
	"fmt"
	"github.com/spf13/cast"
	"io"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"unicode/utf8"
)

func newRouter() *Router {
	return &Router{
		handlerMapper:     make(map[interface{}]HandlerFunc),
		middlewaresMapper: make(map[interface{}][]MiddlewareFunc),
	}
}

// Router is a router for incoming message.
// Router routes the message to its handler and middlewares.
type Router struct {
	// handlerMapper maps message's ID to handler.
	// Handler will be called around middlewares.
	handlerMapper map[interface{}]HandlerFunc

	// middlewaresMapper maps message's ID to a list of middlewares.
	// These middlewares will be called before the handler in handlerMapper.
	middlewaresMapper map[interface{}][]MiddlewareFunc

	// globalMiddlewares is a list of MiddlewareFunc.
	// globalMiddlewares will be called before the ones in middlewaresMapper.
	globalMiddlewares []MiddlewareFunc

	notFoundHandler HandlerFunc
}

// HandlerFunc is the function type for handlers.
type HandlerFunc func(ctx Context)

// MiddlewareFunc is the function type for middlewares.
// A common pattern is like:
//
//	var mf MiddlewareFunc = func(next HandlerFunc) HandlerFunc {
//		return func(ctx Context) {
//			next(ctx)
//		}
//	}
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

var nilHandler HandlerFunc = func(ctx Context) {}

// handleRequest walks ctx through middlewares and handler,
// and returns response message.
func (r *Router) handleRequest(ctx Context) {
	reqMsg := ctx.Request()
	if reqMsg == nil {
		return
	}
	var handler HandlerFunc
	if v, has := r.handlerMapper[reqMsg.ID()]; has {
		handler = v
	}

	var mws = r.globalMiddlewares
	if v, has := r.middlewaresMapper[reqMsg.ID()]; has {
		mws = append(mws, v...) // append to global ones
	}

	// create the handlers stack
	wrapped := r.wrapHandlers(handler, mws)

	// and call the handlers stack
	wrapped(ctx)
}

// wrapHandlers wraps handler and middlewares into a right order call stack.
// Makes something like:
//
//	var wrapped HandlerFunc = m1(m2(m3(handle)))
func (r *Router) wrapHandlers(handler HandlerFunc, middles []MiddlewareFunc) (wrapped HandlerFunc) {
	if handler == nil {
		handler = r.notFoundHandler
	}
	if handler == nil {
		handler = nilHandler
	}
	wrapped = handler
	for i := len(middles) - 1; i >= 0; i-- {
		m := middles[i]
		wrapped = m(wrapped)
	}
	return wrapped
}

// register stores handler and middlewares for id.
func (r *Router) register(id interface{}, h HandlerFunc, m ...MiddlewareFunc) {
	if h != nil {
		r.handlerMapper[id] = h
	}
	ms := make([]MiddlewareFunc, 0, len(m))
	for _, mm := range m {
		if mm != nil {
			ms = append(ms, mm)
		}
	}
	if len(ms) != 0 {
		r.middlewaresMapper[id] = ms
	}
}

// registerMiddleware stores the global middlewares.
func (r *Router) registerMiddleware(m ...MiddlewareFunc) {
	for _, mm := range m {
		if mm != nil {
			r.globalMiddlewares = append(r.globalMiddlewares, mm)
		}
	}
}

// printHandlers prints registered route handlers to console.
func (r *Router) printHandlers(addr string) {
	var w io.Writer = os.Stdout

	_, _ = fmt.Fprintf(w, "\n[EASYTCP] Message-Route Table:\n")
	rows := [][]string{{"Message ID", "Route Handler", "Middleware"}}

	// sort ids
	ids := make([]interface{}, 0, len(r.handlerMapper))
	for id := range r.handlerMapper {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := cast.ToString(ids[i]), cast.ToString(ids[j])
		return a < b
	})

	// add table row
	for _, id := range ids {
		// route handler
		h := r.handlerMapper[id]
		handlerName := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()

		middlewareNames := make([]string, 0, len(r.globalMiddlewares)+len(r.middlewaresMapper[id]))
		// global middleware
		for _, m := range r.globalMiddlewares {
			middlewareName := fmt.Sprintf("%s(g)", runtime.FuncForPC(reflect.ValueOf(m).Pointer()).Name())
			middlewareNames = append(middlewareNames, middlewareName)
		}

		// route middleware
		for _, m := range r.middlewaresMapper[id] {
			middlewareName := runtime.FuncForPC(reflect.ValueOf(m).Pointer()).Name()
			middlewareNames = append(middlewareNames, middlewareName)
		}

		rows = append(rows, []string{fmt.Sprintf("%v", id), handlerName, strings.Join(middlewareNames, "\n")})
	}

	renderTable(w, rows)
	_, _ = fmt.Fprintf(w, "[EASYTCP] Serving at: %s\n\n", addr)
}

func (r *Router) setNotFoundHandler(handler HandlerFunc) {
	r.notFoundHandler = handler
}

// renderTable writes rows as an ASCII table with a header (rows[0]) and a line between rows.
// Cells may contain "\n". The first two columns are right-aligned, the rest left-aligned.
func renderTable(w io.Writer, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	cols := len(rows[0])
	cells := make([][][]string, len(rows)) // row -> col -> lines
	widths := make([]int, cols)
	for i, row := range rows {
		cells[i] = make([][]string, cols)
		for j := 0; j < cols; j++ {
			var text string
			if j < len(row) {
				text = row[j]
			}
			lines := strings.Split(text, "\n")
			cells[i][j] = lines
			for _, l := range lines {
				if n := utf8.RuneCountInString(l); n > widths[j] {
					widths[j] = n
				}
			}
		}
	}

	var sep strings.Builder
	sep.WriteString("+")
	for _, n := range widths {
		sep.WriteString(strings.Repeat("-", n+2))
		sep.WriteString("+")
	}
	sep.WriteString("\n")

	_, _ = io.WriteString(w, sep.String())
	for i := range cells {
		height := 1
		for _, c := range cells[i] {
			if len(c) > height {
				height = len(c)
			}
		}
		for k := 0; k < height; k++ {
			var line strings.Builder
			line.WriteString("|")
			for j := 0; j < cols; j++ {
				var text string
				if k < len(cells[i][j]) {
					text = cells[i][j][k]
				}
				pad := strings.Repeat(" ", widths[j]-utf8.RuneCountInString(text))
				if j < 2 && i > 0 {
					line.WriteString(" " + pad + text + " |")
				} else {
					line.WriteString(" " + text + pad + " |")
				}
			}
			line.WriteString("\n")
			_, _ = io.WriteString(w, line.String())
		}
		_, _ = io.WriteString(w, sep.String())
	}
}
