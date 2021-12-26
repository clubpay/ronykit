package rest

import (
	"sort"
	"strings"

	"github.com/ronaksoft/ronykit"
)

const (
	// paramStart is the character, as a string, which a path pattern starts to define its named parameter.
	paramStart = ":"
	// wildcardParamStart is the character, as a string, which a path pattern starts to define
	// its named parameter for wildcards. It allows everything else after that path prefix
	// but the trie checks for static paths and named parameters before that in order to
	// support everything that other implementations do not, and if nothing else found then it tries to
	//find the closest wildcard path(super and unique).
	wildcardParamStart = "*"
)

type nodeData struct {
	handlers []ronykit.Handler
	decoder  DecoderFunc
}

func (r *rest) route(method, path string, ps ParamsSetter) (*nodeData, error) {
	route := r.routes[method]
	if route == nil {
		return nil, ErrRouteNotFound
	}

	n := route.Search(path, ps)
	if n == nil {
		return nil, ErrRouteNotFound
	}

	return n.Data, nil
}

func (r *rest) Set(method, path string, decoder DecoderFunc, handlers ...ronykit.Handler) {
	method = strings.ToUpper(method)
	if r.routes == nil {
		r.routes = make(map[string]*trie)
	}
	if _, ok := r.routes[method]; !ok {
		r.routes[method] = &trie{
			root:            newTrieNode(),
			hasRootWildcard: false,
		}
	}
	r.routes[method].Insert(
		path, WithTag(method),
		WithHandler(
			&nodeData{
				handlers: handlers,
				decoder:  decoder,
			},
		),
	)
}

// trie contains the main logic for adding and searching nodes for path segments.
// It supports wildcard and named path parameters.
// trie supports very coblex and useful path patterns for routes.
// The trie checks for static paths(path without : or *) and named parameters before that in order to
// support everything that other implementations do not, and if nothing else found then it tries
// to find the closest wildcard path(super and unique).
type trie struct {
	root *trieNode

	// if true then it will handle any path if not other parent wildcard exists,
	// so even 404 (on http services) is up to it, see trie#Insert.
	hasRootWildcard bool

	hasRootSlash bool
}

// InsertOption is just a function which accepts a pointer to a trieNode which can
// alt its `Handler`, `Tag` and `Data`  fields.
// See `WithHandler`, `WithTag` and `WithData`.
type InsertOption func(*trieNode)

// WithHandler sets the node's `Handler` field (useful for HTTP).
func WithHandler(data *nodeData) InsertOption {
	if data == nil {
		panic("data is empty")
	}

	return func(n *trieNode) {
		if n.Data == nil {
			n.Data = data
		}
	}
}

// WithTag sets the node's `Tag` field (maybe useful for HTTP).
func WithTag(tag string) InsertOption {
	return func(n *trieNode) {
		if n.Tag == "" {
			n.Tag = tag
		}
	}
}

// Insert adds a node to the trie.
func (t *trie) Insert(pattern string, options ...InsertOption) {
	if pattern == "" {
		panic("muxie/trie#Insert: empty pattern")
	}

	n := t.insert(pattern, "", nil)
	for _, opt := range options {
		opt(n)
	}
}

const (
	pathSep  = "/"
	pathSepB = '/'
)

func slowPathSplit(path string) []string {
	if path == pathSep {
		return []string{pathSep}
	}

	// remove last sep if any.
	if path[len(path)-1] == pathSepB {
		path = path[:len(path)-1]
	}

	return strings.Split(path, pathSep)[1:]
}

func resolveStaticPart(key string) string {
	i := strings.Index(key, paramStart)
	if i == -1 {
		i = strings.Index(key, wildcardParamStart)
	}
	if i == -1 {
		i = len(key)
	}

	return key[:i]
}

func (t *trie) insert(key, tag string, data *nodeData) *trieNode {
	input := slowPathSplit(key)

	n := t.root
	if key == pathSep {
		t.hasRootSlash = true
	}

	var paramKeys []string

	for _, s := range input {
		c := s[0]

		if isParam, isWildcard := c == paramStart[0], c == wildcardParamStart[0]; isParam || isWildcard {
			n.hasDynamicChild = true
			paramKeys = append(paramKeys, s[1:]) // without : or *.

			// if node has already a wildcard, don't force a value, check for true only.
			if isParam {
				n.childNamedParameter = true
				s = paramStart
			}

			if isWildcard {
				n.childWildcardParameter = true
				s = wildcardParamStart
				if t.root == n {
					t.hasRootWildcard = true
				}
			}
		}

		if !n.hasChild(s) {
			child := newTrieNode()
			n.addChild(s, child)
		}

		n = n.getChild(s)
	}

	n.Tag = tag
	n.Data = data

	n.paramKeys = paramKeys
	n.key = key
	n.staticKey = resolveStaticPart(key)
	n.end = true

	return n
}

// Search is the most important part of the trie.
// It will try to find the responsible node for a specific query (or a request path for HTTP endpoints).
//
// Search supports searching for static paths(path without : or *) and paths that contain
// named parameters or wildcards.
// Priority as:
// 1. static paths
// 2. named parameters with ":"
// 3. wildcards
// 4. closest wildcard if not found, if any
// 5. root wildcard
func (t *trie) Search(q string, params ParamsSetter) *trieNode {
	end := len(q)

	if end == 0 || (end == 1 && q[0] == pathSepB) {
		// fixes only root wildcard but no / registered at.
		if t.hasRootSlash {
			return t.root.getChild(pathSep)
		} else if t.hasRootWildcard {
			// no need to going through setting parameters, this one has not but it is wildcard.
			return t.root.getChild(wildcardParamStart)
		}

		return nil
	}

	n := t.root
	start := 1
	i := 1
	var paramValues []string

	for {
		if i == end || q[i] == pathSepB {
			if child := n.getChild(q[start:i]); child != nil {
				n = child
			} else if n.childNamedParameter { // && n.childWildcardParameter == false {
				n = n.getChild(paramStart)
				if ln := len(paramValues); cap(paramValues) > ln {
					paramValues = paramValues[:ln+1]
					paramValues[ln] = q[start:i]
				} else {
					paramValues = append(paramValues, q[start:i])
				}
			} else if n.childWildcardParameter {
				n = n.getChild(wildcardParamStart)
				if ln := len(paramValues); cap(paramValues) > ln {
					paramValues = paramValues[:ln+1]
					paramValues[ln] = q[start:]
				} else {
					paramValues = append(paramValues, q[start:])
				}

				break
			} else {
				n = n.findClosestParentWildcardNode()
				if n != nil {
					// means that it has :param/static and *wildcard, we go trhough the :param
					// but the next path segment is not the /static, so go back to *wildcard
					// instead of not found.
					//
					// Fixes:
					// /hello/*p
					// /hello/:p1/static/:p2
					// req: http://localhost:8080/hello/dsadsa/static/dsadsa => found
					// req: http://localhost:8080/hello/dsadsa => but not found!
					// and
					// /second/wild/*p
					// /second/wild/static/otherstatic/
					// req: /second/wild/static/otherstatic/random => but not found!
					params.Set(n.paramKeys[0], q[len(n.staticKey):])

					return n
				}

				return nil
			}

			if i == end {
				break
			}

			i++
			start = i

			continue
		}

		i++
	}

	if n == nil || !n.end {
		if n != nil { // we need it on both places, on last segment (below) or on the first unnknown (above).
			if n = n.findClosestParentWildcardNode(); n != nil {
				params.Set(n.paramKeys[0], q[len(n.staticKey):])

				return n
			}
		}

		if t.hasRootWildcard {
			// that's the case for root wildcard, tests are passing
			// even without it but stick with it for reference.
			// Note ote that something like:
			// Routes: /other2/*myparam and /other2/static
			// Reqs: /other2/staticed will be handled
			// by the /other2/*myparam and not the root wildcard (see above), which is what we want.
			n = t.root.getChild(wildcardParamStart)
			params.Set(n.paramKeys[0], q[1:])

			return n
		}

		return nil
	}

	for i, paramValue := range paramValues {
		if len(n.paramKeys) > i {
			params.Set(n.paramKeys[i], paramValue)
		}
	}

	return n
}

// trieNode is the trie's node which path patterns with their data like an HTTP handler are saved to.
// See `trie` too.
type trieNode struct {
	parent *trieNode

	children               map[string]*trieNode
	hasDynamicChild        bool // does one of the children contains a parameter or wildcard?
	childNamedParameter    bool // is the child a named parameter (single segmnet)
	childWildcardParameter bool // or it is a wildcard (can be more than one path segments) ?

	paramKeys []string // the param keys without : or *.
	end       bool     // it is a complete node, here we stop, and we can say that the node is valid.
	key       string   // if end == true then key is filled with the original value of the insertion's key.
	// if key != "" && its parent has childWildcardParameter == true,
	// we need it to track the static part for the closest-wildcard's parameter storage.
	staticKey string

	// insert main data relative to http and a tag for things like route names.
	Data *nodeData
	Tag  string
}

// newTrieNode returns a new, empty, trieNode.
func newTrieNode() *trieNode {
	n := new(trieNode)

	return n
}

func (n *trieNode) addChild(s string, child *trieNode) {
	if n.children == nil {
		n.children = make(map[string]*trieNode)
	}

	if _, exists := n.children[s]; exists {
		return
	}

	child.parent = n
	n.children[s] = child
}

func (n *trieNode) getChild(s string) *trieNode {
	if n.children == nil {
		return nil
	}

	return n.children[s]
}

func (n *trieNode) hasChild(s string) bool {
	return n.getChild(s) != nil
}

func (n *trieNode) findClosestParentWildcardNode() *trieNode {
	n = n.parent
	for n != nil {
		if n.childWildcardParameter {
			return n.getChild(wildcardParamStart)
		}

		n = n.parent
	}

	return nil
}

// keysSorter is the type definition for the sorting logic
// that caller can pass on `GetKeys` and `Autocomplete`.
type keysSorter = func(list []string) func(i, j int) bool

// Keys returns this node's key (if it's a final path segment)
// and its children's node's key. The "sorter" can be optionally used to sort the result.
func (n *trieNode) Keys(sorter keysSorter) (list []string) {
	if n == nil {
		return
	}

	if n.end {
		list = append(list, n.key)
	}

	if n.children != nil {
		for _, child := range n.children {
			list = append(list, child.Keys(sorter)...)
		}
	}

	if sorter != nil {
		sort.Slice(list, sorter(list))
	}

	return
}

// Parent returns the parent of that node, can return nil if this is the root node.
func (n *trieNode) Parent() *trieNode {
	return n.parent
}

// String returns the key, which is the path pattern for the HTTP Mux.
func (n *trieNode) String() string {
	return n.key
}

// IsEnd returns true if this trieNode is a final path, has a key.
func (n *trieNode) IsEnd() bool {
	return n.end
}
