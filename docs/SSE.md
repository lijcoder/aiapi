# Server-Sent Events (SSE) 协议详解

## 什么是 SSE？

Server-Sent Events (SSE) 是一种服务器推送技术，允许服务器主动向客户端发送实时数据。与 WebSocket 不同，SSE 是单向通信，只能从服务器向客户端发送数据。

## 核心特性

### 1. 单向通信
- 只能从服务器向客户端发送数据
- 客户端通过 HTTP 请求建立连接
- 服务器保持连接并持续发送数据

### 2. 自动重连
- 当连接断开时，客户端会自动尝试重新连接
- 可以通过 `retry` 字段设置重连间隔
- 服务端可以控制重连时间

### 3. 事件格式
```
data: 消息内容

data: 第一行
data: 第二行

id: 事件ID
event: 事件类型
data: 事件数据
```

### 4. 多事件类型
- 支持不同类型的事件 (`event` 字段)
- 客户端可以监听特定类型的事件
- 默认事件类型为 `message`

## 工作原理

### 1. 建立连接
```javascript
const eventSource = new EventSource('/api/stream');
```

### 2. 数据传输
- 使用 MIME 类型：`text/event-stream`
- 保持 HTTP 连接持久化
- 实时传输文本数据

### 3. 连接管理
- 浏览器自动处理连接生命周期
- 支持心跳检测
- 自动重连机制

## 与 WebSocket 对比

| 特性 | SSE | WebSocket |
|------|-----|-----------|
| 通信方向 | 单向 (服务器→客户端) | 双向 |
| 协议 | HTTP | 独立协议 |
| 自动重连 | 支持 | 需要手动实现 |
| 连接数限制 | 较少 | 较多 |
| 二进制数据 | 不支持 | 支持 |
| 代理友好 | 是 | 可能需要特殊配置 |

## 使用场景

### 1. 实时通知
```javascript
// 服务器端推送系统通知
eventSource.addEventListener('notification', (event) => {
    const data = JSON.parse(event.data);
    showNotification(data.message);
});
```

### 2. 进度更新
- 文件上传进度
- 数据处理进度
- 长时间运行的任务状态

### 3. 数据流
- 股票价格更新
- 日志流
- 实时监控数据

### 4. 聊天应用（单向）
- 系统消息推送
- 机器人回复

## 实现示例

### 客户端实现
```javascript
// 创建 SSE 连接
const eventSource = new EventSource('/api/events');

// 监听默认事件
eventSource.onmessage = (event) => {
    console.log('收到消息:', event.data);
};

// 监听自定义事件
eventSource.addEventListener('custom', (event) => {
    console.log('自定义事件:', event.data);
});

// 监听连接状态
eventSource.onopen = () => {
    console.log('连接已建立');
};

eventSource.onerror = (error) => {
    console.error('连接错误:', error);
};

// 手动关闭连接
// eventSource.close();
```

### 服务器端实现

#### Go + net/http
```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Data represents the data structure to send
type Data struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// SSEHandler handles Server-Sent Events connections
func SSEHandler(w http.ResponseWriter, r *http.Request) {
	// Set SSE response headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Flush headers immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection comment
	fmt.Fprintf(w, ": 连接已建立\n\n")
	flusher.Flush()

	// Create a channel to handle client disconnect
	clientGone := r.Context().Done()

	// Send periodic data
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			// Client disconnected
			log.Println("Client disconnected")
			return
		case <-ticker.C:
			data := Data{
				Timestamp: time.Now(),
				Message:   "Hello from Go server",
			}
			
			jsonData, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error marshaling JSON: %v", err)
				continue
			}

			// Send data in SSE format
			fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
			flusher.Flush()
		}
	}
}

// CustomEventHandler demonstrates sending custom events
func CustomEventHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, ": Custom event stream started\n\n")
	flusher.Flush()

	clientGone := r.Context().Done()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			log.Println("Client disconnected from custom events")
			return
		case <-ticker.C:
			// Send custom event
			eventData := map[string]interface{}{
				"type":    "progress",
				"percent": time.Now().Unix() % 100,
				"status":  "processing",
			}
			
			jsonData, _ := json.Marshal(eventData)
			
			// Send custom event with specific type
			fmt.Fprintf(w, "event: progress\n")
			fmt.Fprintf(w, "id: %d\n", time.Now().Unix())
			fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
			flusher.Flush()
		}
	}
}

func main() {
	http.HandleFunc("/api/events", SSEHandler)
	http.HandleFunc("/api/custom-events", CustomEventHandler)
	
	fmt.Println("SSE server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

#### Go + Gin 框架
```go
package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ProgressData represents progress update data
type ProgressData struct {
	Stage    string    `json:"stage"`
	Percent  int       `json:"percent"`
	ETA      time.Time `json:"eta"`
	Message  string    `json:"message"`
}

// SSEHandlerUsingGin handles SSE with Gin framework
func SSEHandlerUsingGin(c *gin.Context) {
	// Set headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Get response writer
	w := c.Writer
	
	// Flush headers immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming unsupported",
		})
		return
	}

	// Send initial connection
	fmt.Fprintf(w, ": Gin SSE connection established\n\n")
	flusher.Flush()

	// Create channels for graceful shutdown
	clientGone := c.Request.Context().Done()
	ticker := time.NewTicker(2 * time.Second)
	
	// Start progress simulation
	go func() {
		progress := 0
		for {
			select {
			case <-clientGone:
				return
			case <-ticker.C:
				progress = (progress + 10) % 100
				
				progressData := ProgressData{
					Stage:    "Processing...",
					Percent:  progress,
					ETA:      time.Now().Add(time.Duration(100-progress) * 100 * time.Millisecond),
					Message:  "Progress update",
				}
				
				jsonData, err := json.Marshal(progressData)
				if err != nil {
					continue
				}

				// Send progress update
				fmt.Fprintf(w, "event: progress\n")
				fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
				flusher.Flush()
			}
		}
	}()

	// Keep connection open
	for {
		select {
		case <-clientGone:
			return
		default:
			// Keep connection alive with heartbeat
			fmt.Fprintf(w, ": heartbeat %d\n\n", time.Now().Unix())
			flusher.Flush()
			time.Sleep(30 * time.Second)
		}
	}
}

func main() {
	r := gin.Default()
	
	r.GET("/api/stream", SSEHandlerUsingGin)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	
	r.Run(":8080")
}
```

#### Go + Gorilla WebSocket (高级示例)
```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

// Broadcast to all connected clients
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var clients = make(map[*Client]bool)
var broadcast = make(chan []byte, 100)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	// Create new client
	client := &Client{
		conn: ws,
		send: make(chan []byte, 100),
	}
	clients[client] = true

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	// Send welcome message
	welcome := map[string]string{
		"type":    "welcome",
		"message": "Connected to WebSocket server",
		"server":  "Go WebSocket Server",
	}
	welcomeJSON, _ := json.Marshal(welcome)
	client.send <- welcomeJSON
}

func (c *Client) readPump() {
	defer func() {
		delete(clients, c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// Echo message back (optional)
		c.send <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send regular message
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Broadcast queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func broadcastMessages() {
	for {
		select {
		case message := <-broadcast:
			for client := range clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(clients, client)
					client.conn.Close()
				}
			}
		}
	}
}

func simulateData() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			data := map[string]interface{}{
				"timestamp": time.Now().Unix(),
				"data":      "Random data",
				"id":        time.Now().Format("20060102150405"),
			}
			jsonData, _ := json.Marshal(data)
			broadcast <- jsonData
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)

	go broadcastMessages()
	go simulateData()

	log.Println("WebSocket server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 协议格式详解

### 1. 数据行
```
data: 实际数据内容
```

### 2. 事件类型
```
event: customEvent
data: 自定义事件数据
```

### 3. 事件ID
```
id: 123
data: 带ID的事件数据
```

### 4. 重连时间
```
retry: 10000
data: 设置重连间隔为10秒
```

### 5. 注释行
```
: 这是一条注释，不会发送到客户端
```

### 6. 空行
```
data: 第一条消息

data: 第二条消息
```

## 优缺点分析

### 优点
1. **简单易用**：API 简洁，易于实现
2. **自动重连**：内置重连机制
3. **HTTP 友好**：基于 HTTP，代理和防火墙兼容性更好
4. **事件支持**：支持多事件类型和事件 ID
5. **性能开销小**：相对 WebSocket 资源消耗更少

### 缺点
1. **单向通信**：只能服务器推送到客户端
2. **连接数限制**：浏览器限制同域连接数
3. **文本传输**：只能传输文本数据
4. **连接超时**：需要心跳维持长连接
5. **不支持二进制**：只能发送字符串数据

## 浏览器兼容性

| 浏览器 | 支持版本 |
|--------|----------|
| Chrome | 6+ |
| Firefox | 6+ |
| Safari | 5+ |
| Edge | 12+ |
| Opera | 11+ |

## 最佳实践

### 1. 错误处理
```javascript
eventSource.onerror = (error) => {
    // 实现重试逻辑
    console.error('SSE 连接错误:', error);
};
```

### 2. 连接管理
```javascript
// 检查连接状态
if (eventSource.readyState === EventSource.CONNECTING) {
    console.log('正在连接...');
} else if (eventSource.readyState === EventSource.OPEN) {
    console.log('连接已建立');
} else if (eventSource.readyState === EventSource.CLOSED) {
    console.log('连接已关闭');
}
```

### 3. 内存管理
```javascript
// 页面卸载时关闭连接
window.addEventListener('beforeunload', () => {
    eventSource.close();
});
```

### 4. 安全考虑
```javascript
// 使用 HTTPS
const eventSource = new EventSource('https://api.example.com/events');

// 添加认证头（需要服务器支持）
const eventSource = new EventSource('/api/events', {
    withCredentials: true
});
```

## 常见问题解决

### 1. 连接频繁断开
- 检查网络稳定性
- 设置合适的重连时间
- 实现心跳机制

### 2. 性能问题
- 控制消息发送频率
- 避免发送过大数据
- 及时清理无效连接

### 3. 代理缓存问题
```javascript
// 服务器端设置
res.setHeader('Cache-Control', 'no-cache');
res.setHeader('Connection', 'keep-alive');
```

## Go 特定的注意事项

### 1. 依赖管理
```bash
# 使用 Go modules
go mod init sse-server

# 添加依赖
go get github.com/gin-gonic/gin
go get github.com/gorilla/websocket
```

### 2. 并发处理
Go 的 goroutine 特性非常适合处理 SSE 连接：
```go
// 为每个客户端启动一个 goroutine
for client := range clients {
    go func(c *Client) {
        // 处理客户端连接
    }(client)
}
```

### 3. 资源管理
```go
// 确保在连接断开时清理资源
defer ticker.Stop()
defer close(client.send)
```

## 总结

SSE 是一个轻量级的实时通信解决方案，特别适合以下场景：

- 单向实时数据推送
- 简单的事件通知系统
- 进度更新和状态监控
- 对 WebSocket 功能需求不复杂的应用

虽然功能相对有限，但 SSE 的简单性和 HTTP 兼容性使其在很多场景下成为 WebSocket 的良好替代方案。Go 语言的并发特性和简洁的语法使其成为实现 SSE 服务器的理想选择。

选择 SSE 还是 WebSocket，应该根据具体的需求和技术架构来决定。