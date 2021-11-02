package zinc

import (
	"fmt"
	"strings"
)

// node 前缀树节点
type node struct {
	pattern 	string	// 要么是一个完整的url，要么是一个空字符串
	part    	string	// URL块值，用/分割的部分，比如/abc/123中，abc和123就是2个part
	children 	[]*node	// 当前节点下的子节点
	isWild		bool	// 是否模糊匹配，比如:filename或*filename这样的node就为true
}

func (n *node) String() string {
	return fmt.Sprintf("node{pattern=%s, part=%s, isWild=%t}", n.pattern, n.part, n.isWild)
}

// matchChild 方法返回第一个匹配成功的节点，用于insert插入方法中
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		// 修改点：动态匹配做强校验,防止路由注册时被覆盖
		if child.part == part || ((part[0] == ':' || part[0] == '*') && child.isWild) {
			return child
		}
	}
	return nil
}

// matchChildren 方法返回所有匹配成功的节点，用于search查找方法中
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	wildNodes := make([]*node, 0)
	for _, child := range n.children {
		// 修改点：静态路由节点优先,动态路由节点延后
		if child.part == part {
			nodes = append(nodes, child)
		} else if child.isWild {
			wildNodes = append(wildNodes, child)
		}
	}
	nodes = append(nodes, wildNodes...)
	return nodes
}

// insert 方法一边匹配一边插入，pattern为完整url，parts为url各部分，height是当前层高（初始为0）
func (n *node) insert(pattern string, parts []string, height int) {
	// 递归的终止条件
	if len(parts) == height {
		// 如果已经匹配完了，那么将pattern赋值给该node，表示它是一个完整的url
		n.pattern = pattern
		return
	}

	part := parts[height]
	child := n.matchChild(part)
	if child == nil {
		// 没有匹配上，那么生成新节点，并放到n节点的子列表中
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	// 接着插入下一个part节点
	child.insert(pattern, parts, height+1)
}

// search 方法查找匹配的route（返回的node中pattern为完整url)
func (n *node) search(parts []string, height int) *node {
	// 递归终止条件，找到末尾了或者通配符
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		// pattern为空字符串表示它不是一个完整的url，匹配失败
		if n.pattern == "" {
			return nil
		}
		return n
	}

	part := parts[height]
	// 获取所有可能的子路径
	children := n.matchChildren(part)

	for _, child := range children {
		// 对于每条路径接着用下一part去查找
		result := child.search(parts, height+1)
		if result != nil {
			// 找到了即返回
			return result
		}
	}

	return nil
}

// travel 方法查找所有完整的url，保存到列表中
func (n *node) travel(list *([]*node)) {
	// 递归终止条件
	if n.pattern != "" {
		*list = append(*list, n)
	}
	for _, child := range n.children {
		// 一层一层的递归找pattern是非空的节点
		child.travel(list)
	}
}