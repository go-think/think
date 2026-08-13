package router

import (
	"strings"
)

type nodeType uint8

const (
	static nodeType = iota
	root
	param
	catchAll
)

type node struct {
	path      string
	indices   string
	wildChild bool
	nType     nodeType
	priority  uint32
	children  []*node
	handle    interface{}
	paramName string
}

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, returned by the router.
type Params []Param

// Get returns the value of the first Param which key matches the given name.
func (ps Params) Get(name string) (string, bool) {
	for _, entry := range ps {
		if entry.Key == name {
			return entry.Value, true
		}
	}
	return "", false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func countParams(path string) uint16 {
	var n uint16
	for i := 0; i < len(path); i++ {
		if path[i] == ':' || path[i] == '*' || path[i] == '{' {
			n++
		}
	}
	return n
}

// addRoute adds a node with the given handle to the path.
func (n *node) addRoute(path string, handle interface{}) {
	fullPath := path
	n.priority++

	// Empty tree
	if len(n.path) == 0 && len(n.children) == 0 {
		n.insertChild(path, fullPath, handle)
		n.nType = root
		return
	}

walk:
	for {
		// Find the longest common prefix.
		i := min(len(path), len(n.path))
		for i > 0 && path[:i] != n.path[:i] {
			i--
		}

		// Split edge
		if i < len(n.path) {
			child := node{
				path:      n.path[i:],
				wildChild: n.wildChild,
				nType:     static,
				indices:   n.indices,
				children:  n.children,
				handle:    n.handle,
				priority:  n.priority - 1,
			}

			n.children = []*node{&child}
			n.indices = string([]byte{n.path[i]})
			n.path = path[:i]
			n.handle = nil
			n.wildChild = false
		}

		// Make new node a child of this node
		if i < len(path) {
			path = path[i:]

			if n.wildChild {
				n = n.children[0]
				n.priority++

				// Check if the path matches the wildChild.
				if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
					// Check for longer wildcard, e.g. :name and :names
					if len(n.path) >= len(path) || path[len(n.path)] == '/' {
						continue walk
					}
				}
				panic("path segment '" + path + "' conflicts with existing wildcard in path '" + fullPath + "'")
			}

			c := path[0]

			// slash after param
			if n.nType == param && c == '/' && len(n.children) == 1 {
				n = n.children[0]
				n.priority++
				continue walk
			}

			// Check if a child with the next path byte exists
			for j, max := 0, len(n.indices); j < max; j++ {
				if c == n.indices[j] {
					j = n.incrementChildPriority(j)
					n = n.children[j]
					continue walk
				}
			}

			// Otherwise insert child
			if c != ':' && c != '*' && c != '{' {
				n.indices += string([]byte{c})
				child := &node{}
				n.children = append(n.children, child)
				n.incrementChildPriority(len(n.indices) - 1)
				n = child
			}
			n.insertChild(path, fullPath, handle)
			return
		}

		// Otherwise add handle to current node
		n.handle = handle
		return
	}
}

func (n *node) insertChild(path string, fullPath string, handle interface{}) {
	for {
		// Find prefix until first wildcard
		wildcard, paramName, valid := findWildcard(path)
		if !valid {
			break
		}

		// The wildcard name must not contain ':' and '*'
		if len(paramName) < 1 {
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		if len(n.children) > 0 {
			panic("wildcard route '" + wildcard + "' conflicts with existing children in path '" + fullPath + "'")
		}

		if wildcard[0] == ':' || wildcard[0] == '{' { // param
			if len(path) > 0 {
				// Insert prefix before the current wildcard
				n.path = path[:strings.Index(path, wildcard)]
				path = path[len(n.path):]
			}

			child := &node{
				nType:     param,
				path:      wildcard,
				paramName: paramName,
			}
			n.children = []*node{child}
			n.wildChild = true
			n = child
			n.priority++

			// If the path doesn't end with the wildcard, then there
			// will be another non-wildcard subpath starting with '/'
			if len(wildcard) < len(path) {
				path = path[len(wildcard):]
				child := &node{
					priority: 1,
				}
				n.children = []*node{child}
				n = child
				continue
			}

			n.handle = handle
			return
		}

		// catchAll (*)
		if len(path) > 0 {
			n.path = path[:strings.Index(path, wildcard)]
			path = path[len(n.path):]
		}

		child := &node{
			path:      wildcard,
			paramName: paramName,
			nType:     catchAll,
			priority:  1,
			handle:    handle,
		}
		n.children = []*node{child}
		n.wildChild = true
		n = child
		return
	}

	// If no wildcard was found, simply insert the path and handle
	n.path = path
	n.handle = handle
}

func findWildcard(path string) (wildcard string, paramName string, valid bool) {
	// Find start
	for start, c := range []byte(path) {
		if c == ':' {
			end := start + 1
			for end < len(path) && path[end] != '/' {
				end++
			}
			return path[start:end], path[start+1 : end], true
		} else if c == '{' {
			end := start + 1
			for end < len(path) && path[end] != '}' {
				end++
			}
			if end < len(path) && path[end] == '}' {
				return path[start : end+1], path[start+1 : end], true
			}
		} else if c == '*' {
			pName := path[start+1:]
			if pName == "" {
				pName = "filepath"
			}
			return path[start:], pName, true
		}
	}
	return "", "", false
}

func (n *node) incrementChildPriority(i int) int {
	children := n.children
	children[i].priority++
	prio := children[i].priority

	// Adjust position (sliding to keep sorted by priority)
	newPos := i
	for ; newPos > 0 && children[newPos-1].priority < prio; newPos-- {
		children[newPos], children[newPos-1] = children[newPos-1], children[newPos]
	}

	// Rebuild indices string
	if newPos != i {
		n.indices = n.indices[:newPos] +
			n.indices[i:i+1] +
			n.indices[newPos:i] +
			n.indices[i+1:]
	}

	return newPos
}

// getValue returns the handle registered to the given path.
func (n *node) getValue(path string) (handle interface{}, ps Params, tsr bool) {
walk:
	for {
		prefix := n.path
		if len(path) > len(prefix) {
			if path[:len(prefix)] == prefix {
				path = path[len(prefix):]

				// If this node has a wild child, then check child
				if n.wildChild {
					n = n.children[0]
					switch n.nType {
					case param:
						// Find param end (slash or end of path)
						end := 0
						for end < len(path) && path[end] != '/' {
							end++
						}

						// Save param value
						if ps == nil {
							ps = make(Params, 0, 4)
						}
						ps = append(ps, Param{
							Key:   n.paramName,
							Value: path[:end],
						})

						// We need to go deeper!
						if end < len(path) {
							if len(n.children) > 0 {
								path = path[end:]
								n = n.children[0]
								continue walk
							}

							tsr = len(path) == end+1
							return
						}

						if handle = n.handle; handle != nil {
							return
						} else if len(n.children) == 1 {
							// No handle found. Check if a handle for this path + a trailing slash exists
							n = n.children[0]
							tsr = n.path == "/" && n.handle != nil
						}
						return

					case catchAll:
						if ps == nil {
							ps = make(Params, 0, 4)
						}
						ps = append(ps, Param{
							Key:   n.paramName,
							Value: path,
						})

						handle = n.handle
						return

					default:
						panic("invalid node type")
					}
				}

				// Check if a child with the next path byte exists
				c := path[0]
				for i, max := 0, len(n.indices); i < max; i++ {
					if c == n.indices[i] {
						n = n.children[i]
						continue walk
					}
				}

				// Nothing found.
				tsr = (path == "/" && n.handle != nil)
				return
			}
		} else if path == prefix {
			// We should have reached the node containing the handle.
			if handle = n.handle; handle != nil {
				return
			}

			// If there is no handle for this route, but this route has a
			// wildcard child, there must be a handle for this path with an additional slash
			if path == "/" && n.wildChild && n.nType != root {
				tsr = true
				return
			}

			for i, max := 0, len(n.indices); i < max; i++ {
				if n.indices[i] == '/' {
					n = n.children[i]
					tsr = (len(n.path) == 1 && n.handle != nil) ||
						(n.nType == catchAll && n.children[0].handle != nil)
					return
				}
			}

			return
		}

		// Nothing found.
		tsr = (path == "/") ||
			(len(prefix) == len(path)+1 && prefix[len(path)] == '/' && path == prefix[:len(path)] && n.handle != nil)
		return
	}
}
