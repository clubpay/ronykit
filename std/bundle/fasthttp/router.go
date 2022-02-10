package fasthttp

import (
	"net/http"
	"strings"
	"sync"

	"github.com/ronaksoft/ronykit"
)

type routeData struct {
	Method      string
	Path        string
	ServiceName string
	Decoder     DecoderFunc
	Handlers    []ronykit.Handler
	Modifiers   []ronykit.Modifier
}

// mux is a http.Handler which can be used to dispatch requests to different
// handler functions via configurable routes.
type mux struct {
	trees map[string]*node

	paramsPool sync.Pool
	maxParams  uint16

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 308 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the mux tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the mux does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the mux makes a redirection
	// to the corrected path with status code 301 for GET requests and 308 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the mux checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	// If enabled, the mux automatically replies to OPTIONS requests.
	// Custom OPTIONS handlers take priority over automatic replies.
	HandleOPTIONS bool

	// An optional http.Handler that is called on automatic OPTIONS requests.
	// The handler is only called if HandleOPTIONS is true and no OPTIONS
	// handler for the specific path was set.
	// The "Allowed" header is set before calling the handler.
	GlobalOPTIONS http.Handler

	// Cached value of global (*) allowed methods
	globalAllowed string

	// Configurable http.Handler which is called when no matching route is
	// found. If it is not set, http.NotFound is used.
	NotFound http.Handler

	// Configurable http.Handler which is called when a request
	// cannot be routed and HandleMethodNotAllowed is true.
	// If it is not set, http.Error with http.StatusMethodNotAllowed is used.
	// The "Allow" header with allowed request methods is set before the handler
	// is called.
	MethodNotAllowed http.Handler

	// Function to handle panics recovered from http handlers.
	// It should be used to generate a error page and return the http error code
	// 500 (Internal Server Error).
	// The handler can be used to keep your server from crashing because of
	// unrecovered panics.
	PanicHandler func(http.ResponseWriter, *http.Request, interface{})
}

func (r *mux) getParams() *Params {
	ps, _ := r.paramsPool.Get().(*Params)
	*ps = (*ps)[0:0] // reset slice

	return ps
}

func (r *mux) putParams(ps *Params) {
	if ps != nil {
		r.paramsPool.Put(ps)
	}
}

// GET is a shortcut for mux.Handle(http.MethodGet, path, handle)
func (r *mux) GET(path string, handle *routeData) {
	r.Handle(http.MethodGet, path, handle)
}

// HEAD is a shortcut for mux.Handle(http.MethodHead, path, handle)
func (r *mux) HEAD(path string, handle *routeData) {
	r.Handle(http.MethodHead, path, handle)
}

// OPTIONS is a shortcut for mux.Handle(http.MethodOptions, path, handle)
func (r *mux) OPTIONS(path string, handle *routeData) {
	r.Handle(http.MethodOptions, path, handle)
}

// POST is a shortcut for mux.Handle(http.MethodPost, path, handle)
func (r *mux) POST(path string, handle *routeData) {
	r.Handle(http.MethodPost, path, handle)
}

// PUT is a shortcut for mux.Handle(http.MethodPut, path, handle)
func (r *mux) PUT(path string, handle *routeData) {
	r.Handle(http.MethodPut, path, handle)
}

// PATCH is a shortcut for mux.Handle(http.MethodPatch, path, handle)
func (r *mux) PATCH(path string, handle *routeData) {
	r.Handle(http.MethodPatch, path, handle)
}

// DELETE is a shortcut for mux.Handle(http.MethodDelete, path, handle)
func (r *mux) DELETE(path string, handle *routeData) {
	r.Handle(http.MethodDelete, path, handle)
}

// Handle registers a new request handle with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *mux) Handle(method, path string, handle *routeData) {
	varsCount := uint16(0)

	if method == "" {
		panic("method must not be empty")
	}
	if len(path) < 1 || path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	if r.trees == nil {
		r.trees = make(map[string]*node)
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root

		r.globalAllowed = r.allowed("*", "")
	}

	root.addRoute(path, handle)

	// Update maxParams
	if paramsCount := countParams(path); paramsCount+varsCount > r.maxParams {
		r.maxParams = paramsCount + varsCount
	}

	// Lazy-init paramsPool alloc func
	if r.paramsPool.New == nil && r.maxParams > 0 {
		r.paramsPool.New = func() interface{} {
			ps := make(Params, 0, r.maxParams)

			return &ps
		}
	}
}

// Lookup allows the manual lookup of a method + path combo.
// This is e.g. useful to build a framework around this mux.
// If the path was found, it returns the handle function and the path parameter
// values. Otherwise the third return value indicates whether a redirection to
// the same path with an extra / without the trailing slash should be performed.
func (r *mux) Lookup(method, path string) (*routeData, Params, bool) {
	if root := r.trees[method]; root != nil {
		handle, ps, tsr := root.getValue(path, r.getParams)
		if handle == nil {
			r.putParams(ps)

			return nil, nil, tsr
		}
		if ps == nil {
			return handle, nil, tsr
		}

		return handle, *ps, tsr
	}

	return nil, nil, false
}

func (r *mux) allowed(path, reqMethod string) (allow string) {
	allowed := make([]string, 0, 9)

	if path == "*" { // server-wide
		// empty method is used for internal calls to refresh the cache
		if reqMethod == "" {
			for method := range r.trees {
				if method == http.MethodOptions {
					continue
				}
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		} else {
			return r.globalAllowed
		}
	} else { // specific path
		for method := range r.trees {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == http.MethodOptions {
				continue
			}

			handle, _, _ := r.trees[method].getValue(path, nil)
			if handle != nil {
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		}
	}

	if len(allowed) > 0 {
		// Add request method to list of allowed methods
		allowed = append(allowed, http.MethodOptions)

		// Sort allowed methods.
		// sort.Strings(allowed) unfortunately causes unnecessary allocations
		// due to allowed being moved to the heap and interface conversion
		for i, l := 1, len(allowed); i < l; i++ {
			for j := i; j > 0 && allowed[j] < allowed[j-1]; j-- {
				allowed[j], allowed[j-1] = allowed[j-1], allowed[j]
			}
		}

		// return as comma separated list
		return strings.Join(allowed, ", ")
	}

	return allow
}
