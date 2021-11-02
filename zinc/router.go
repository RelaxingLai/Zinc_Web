package zinc

import (
	"net/http"
	"strings"
)

// router 路由结构
type router struct {
	// 使用 roots 来存储每种请求方式的Trie 树根节点。
	roots    map[string]*node
	handlers map[string]HandlerFunc
}

// roots key 例子： roots['GET']、roots['POST']
// handlers key 例子： handlers['GET-/p/:lang/doc']、handlers['POST-/p/book']

// newRouter 是 zinc.router 的构造函数
func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// parsePattern 返回一个由给定pattern拆分的多个part组成的切片
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")

	parts := make([]string, 0)
	for _, item := range vs {
		if item != "" {
			parts = append(parts, item)
			// 一个url中最多只能有一个'*'通配符
			if item[0] == '*' {
				break
			}
		}
	}
	return parts
}

// addRoute 方法将pattern和对应处理函数注册到路由表中，并将路由插入到method对应的前缀树中
func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {
	// 拆分pattern（url）
	parts := parsePattern(pattern)

	key := method + "-" + pattern
	// 注册到路由表
	r.handlers[key] = handler

	_, ok := r.roots[method]
	// 该method对应的前缀树不存在，创建根节点
	if !ok {
		r.roots[method] = &node{}
	}
	// 将pattern拆分的part逐个插入method对应的前缀树中
	r.roots[method].insert(pattern, parts, 0)
}

// getRoute 方法取得路由。
// 解析了`:`和`*`两种匹配符的参数；
// 返回path对应的node（已注册的route）和储存解析结果的params（map类型） 。
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	root, ok := r.roots[method]
	// 该method对应的前缀树不存在
	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)

	if n != nil {
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			// 如：`/p/go/doc`匹配到`/p/:lang/doc`，解析结果为：`{lang: "go"}`；
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			// 如：`/static/css/zincRe.css`匹配到`/static/*filepath`，解析结果为`{filepath: "css/zincRe.css"}`。
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}

// getRoutes 方法返回method作为root下的所有route（每一个node即已注册的route)
func (r *router) getRoutes(method string) []*node {
	root, ok := r.roots[method]
	if !ok {
		return nil
	}
	nodes := make([]*node, 0)
	root.travel(&nodes)
	return nodes
}

// handle 方法匹配路由对应的处理函数Handler ，添加到(*Context).handlers列表中；
// 通过传入的 Context对象 的 Next 方法，依次调用 Context对象handlers列表中的Handler和中间件。
//
// handle 方法将解析出来的路由参数赋值给了 Context对象 的 Params
//（如：GET /a/asd/c || GET a/s/c 匹配到路由(GET-/a/:param/c)对应的HandlerFunc，并把asd || s 存在Context的Params里）。
func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)

	if n != nil {
		// 将解析出来的路由参数赋值给了c.Params
		c.Params = params
		key := c.Method + "-" + n.pattern
		// 将从路由匹配得到的 Handler 添加到 `c.handlers`列表中
		c.handlers = append(c.handlers, r.handlers[key])
	} else {
		// 匹配失败，将显示匹配失败的函数添加到 `c.handlers`列表中
		c.handlers = append(c.handlers, func(c *Context) {
			c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
		})
	}

	c.Next()
}