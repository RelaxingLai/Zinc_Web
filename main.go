package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
	"zinc"
)

func onlyForG2() zinc.HandlerFunc {
	return func(c *zinc.Context) {
		// 启动计时器
		t := time.Now()
		// 如果一个 server error 出现
		c.Fail(500, "Internal Server Error")
		// 计算解决时间
		log.Printf("[%d] %s in %v for group g2", c.StatusCode, c.Req.RequestURI, time.Since(t))
	}
}

func main() {
    e := zinc.New()
	// 全局中间件
	e.Use(zinc.Logger(), zinc.Recovery()) 
    
	// 根路径
	e.GET("/", func(c *zinc.Context) {
		c.String(http.StatusOK, "Hello zincRe\n")
	})
	
	// 动态路由
	e.GET("/assets/*filepath", func(c *zinc.Context) {
		c.JSON(http.StatusOK, zinc.H{"filepath": c.Param("filepath")})
	})
	
	// 数组下标越界 测试 Recovery()
	e.GET("/panic", func(c *zinc.Context) {
		names := []string{"zincRe"}
		c.String(http.StatusOK, names[100])           
	})
	
	// g1 分组
	g1 := e.Group("/g1")
	g1.GET("/hello/:name", func(c *zinc.Context) {
		// /hello/zincRe
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
	})
	g1.POST("/login", func(c *zinc.Context) {
		c.JSON(http.StatusOK, zinc.H{
			"username": c.PostForm("username"),
			"password": c.PostForm("password"),
		})
	})
	
	// g2 分组
	g2 := e.Group("/g2")
	g2.Use(onlyForG2()) { // g2 分组中间件
		g2.GET("/hello/:name", func(c *zinc.Context) {
			// /hello/zincRe
			c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
		})
	}

	// 启动HTTP服务
	e.Run(":9999")
}
