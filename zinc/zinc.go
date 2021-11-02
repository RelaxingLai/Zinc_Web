package zinc

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

// HandlerFunc 定义了使用zinc框架时的请求处理函数（handler）
type HandlerFunc func(*Context)

// Engine 对整个zinc框架的所有资源统一协调。
// Engine 通过实现 ServeHTTP 方法，以实现 http.Handler 接口。
type Engine struct {
	*RouterGroup           // 嵌套结构体，继承RouterGroup所有属性和方法
	router *router         // 普通路由结构
	groups []*RouterGroup  // 存储所有分组
	htmlTemplates *template.Template // 将所有的模板加载进内存，用于html渲染
	funcMap       template.FuncMap   // 是所有的自定义模板渲染函数，用于html渲染
}

// RouterGroup 分组路由结构
type RouterGroup struct {
	prefix      string         // 前缀
	middlewares []HandlerFunc  // 中间件
	engine      *Engine        // 所有分组都指向同一个Engine
}

// New 是 zinc.Engine 的构造函数
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// Group 方法创建一个新的RouterGroup。
// 所有分组都指向同一个Engine。
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// Use 方法将中间件应用到 group 分组中
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

//  addRoute 方法把路由（由请求方法和路由地址构成）和处理方法注册到路由映射表 router 中
func (group *RouterGroup) addRoute(method string, comp string, handler HandlerFunc) {
	// 加上分组的前缀 group.prefix 组成 pattern
	pattern := group.prefix + comp
	log.Printf("Route %4s - %s", method, pattern)
	group.engine.router.addRoute(method, pattern, handler)
}

// GET 方法把请求方法为"GET"的请求和相应处理方法 addRoute
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST 方法把请求方法为"POST"的请求和相应处理方法 addRoute
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// createStaticHandler 方法创建静态文件处理器
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	// 路径合并，absolutePath 为 group.prefix/relativePath
	absolutePath := path.Join(group.prefix, relativePath)
	// http.StripPrefix 返回一个处理器fileServer，
	// fileServer 会将请求的 URL.Path 字段中给定前缀 absolutePath 去除后再交由处理器 http.FileServer(fs) 处理。
	// http.FileServer 返回一个处理器，该处理器 使用给定文件系统fs的内容 响应所有HTTP请求。
	//
	// 如：g1/assets/js/zincRe.js 中 absolutePath 为 g1/assets ,则 http.FileServer(fs) 处理 js/zincRe.js；
	//    http.FileServer(fs)中 fs 为文件系统类型的 /usr/zincRe/blog/static；
	//    所以是处理器以文件系统 /usr/zincRe/blog/static 中的内容响应所有前缀为g1/assets的HTTP请求。
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))

	return func(c *Context) {
		file := c.Param("filepath")
		// 检查文件是否存在，是否有权访问它
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		// 调用(http.Handler).ServeHTTP 方法响应HTTP请求。
		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// Static 方法将磁盘上的某个文件夹 root 映射到路由 relativePath，
// 并生成对应静态handler后与 relativePath/*filepath 绑定、注册到分组中。
//
// 如：(*RouterGroup).Static("/assets", "/usr/zincRe/blog/static")；
// 用户访问`/assets/js/zincRe.js`，最终返回`/usr/zincRe/blog/static/js/zincRe.js`。
func (group *RouterGroup) Static(relativePath string, root string) {
	// 创建静态文件处理器 handler
	// http.Dir() 方法会返回 http.Dir 类型用于 将字符串路径转换为文件系统。
	// Dir类型实现了 http.FileSystem 接口。
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	// 路径合并，urlPattern为 relativePath/*filepath。
	urlPattern := path.Join(relativePath, "/*filepath")
	// 注册 GET方法路由，将 relativePath/*filepath 与 handler 绑定。
	group.GET(urlPattern, handler)
}

// SetFuncMap 方法设置自定义渲染函数
func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

// LoadHTMLGlob 方法加载模板
func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}

// Run 方法启动一个 http 服务器
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

// ServeHTTP 方法构造初始化一个Context对象；
// Context对象保存所有适用于当前请求的中间件；
// Context对象作为engine调用router.handle方法的参数。
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// 当前请求适用的中间件列表
	var middlewares []HandlerFunc
	// 遍历所有分组
	for _, group := range engine.groups {
		// 若此 group.prefix 为 URL.Path 的前缀
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			// 当前请求适用于此 group 分组的所有中间件
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, req)
	c.handlers = middlewares
	c.engine = engine
	engine.router.handle(c)
}