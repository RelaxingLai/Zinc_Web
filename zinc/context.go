package zinc

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

// Context 上下文结构
type Context struct {
	// 原始对象
	Writer http.ResponseWriter
	Req *http.Request
	// 请求信息
	Method string            // 请求方法，如：'GET'、'POST'
	Path string              // URL中的路径部分
	Params map[string]string // 解析后的动态路由参数
	// 响应信息
	StatusCode int           // HTTP报文的状态码
	// 中间件
	handlers []HandlerFunc   // 处理函数列表（中间件或Handler）
	index    int             // handlers下标
	// Engine 指针
	engine *Engine           // 用来访问 Engine 中的 HTML 模板
}

// newContext 是 zinc.Context 的构造函数
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Path:   req.URL.Path,
		Method: req.Method,
		Req:    req,
		Writer: w,
		// 初始化为-1
		index:  -1,
	}
}

// Next 方法进入后面的处理函数(中间件或用户定义的Handler)
func (c *Context) Next() {
	c.index++
	handlersLen := len(c.handlers)
	for ; c.index < handlersLen ; c.index++ {
		c.handlers[c.index](c)
	}
}

// Fail 方法作为测试用的短路中间件，用发送500错误码来表示中间件起作用了
func (c *Context) Fail(code int, err string) {
	c.index = len(c.handlers)
	c.JSON(code, H{"message": err})
}

// Param 方法提供对动态路由参数的访问
func (c *Context) Param(key string) string {
	value, _ := c.Params[key]
	return value
}

// PostForm 方法返回c.Req内以key为键映射的表单数据（的第一个值）
func (c *Context) PostForm(key string) string {
	// FormValue返回key为键查询http.Request对象的Form字段得到结果[]string切片的第一个值。
	// Form是url.Values类型，是解析好的表单数据，包括URL字段的query参数和POST或PUT的表单数据。
	// Values类型即map[string][]string类型，将键映射到值的列表。一般用于查询的参数和表单的属性。
	return c.Req.FormValue(key)
}

// Query 方法返回c.Req.URL编码后的查询字符串部分（'?'后‘#’前的部分）中key为键对应的第一个值
func (c *Context) Query(key string) string {
	// c.Req.URL字段是 *url.URL类型，代表一个解析后的URL。
	// Query方法解析URL对象的RawQuery字段（编码后的查询字符串）并返回其表示的Values类型键值对。
	// Values对象的Get方法会获取key对应的值集的第一个值。
	return c.Req.URL.Query().Get(key)
}

// Status 方法设置c中HTTP响应报文的状态码
func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Writer.WriteHeader(code)
}

// SetHeader 方法设置c中HTTP响应报文的头部
func (c *Context) SetHeader(key string,value string) {
	// Header方法返回一个Header类型值，该值会被WriteHeader方法发送。
	// 在调用WriteHeader或Write方法后再改变该Header对象是没有意义的。（所以要先设置）
	// Header类型即map[string][]string类型，代表HTTP头部的键值对。
	c.Writer.Header().Set(key,value)
}

// String 方法快速构造String响应报文
func (c *Context) String(code int,format string,values ...interface{}) {
	// 调用顺序Header().Set，WriteHeader()，Write()
	// 在调用WriteHeader或Write方法后再改变Header对象是没有意义的。
	// 如果WriteHeader没有被显式调用，第一次调用Write时会触发隐式调用WriteHeader(http.StatusOK)
	c.SetHeader("Content-Type","text/plain")
	c.Status(code)
	c.Writer.Write([]byte(fmt.Sprintf(format,values...)))
}

// HTML 方法快速构造HTML响应报文。
func (c *Context) HTML(code int, name string, data interface{}) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	// 根据模板文件名 name 选择模板进行渲染。
	err := c.engine.htmlTemplates.ExecuteTemplate(c.Writer, name, data)
	if err != nil {
		c.Fail(500, err.Error())
	}
}

// JSON 方法快速构造JSON响应报文
func (c *Context) JSON(code int,obj interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	// Encoder类型的作用是将json对象写入输出流。
	// NewEncoder方法创建一个将数据写入输出流的*Encoder。
	encode := json.NewEncoder(c.Writer)
	// Encode方法将obj的json编码写入输出流
	if err := encode.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

// Data 方法快速构造data（[]byte类型）响应报文
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}