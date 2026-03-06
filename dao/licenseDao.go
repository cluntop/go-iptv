package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iptv/dto"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var WS = NewWSClient()
var Lic dto.Lic

// =========================
// 数据结构
// =========================

type Request struct {
	Action string      `json:"a"`
	Data   interface{} `json:"d"`
}

type Response struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// =========================
// WSClient（线程安全版）
// =========================

type wsMessage struct {
	req      Request
	respChan chan Response
	errChan  chan error
}

type WSClient struct {
	url    string
	conn   *websocket.Conn
	rw     sync.RWMutex
	closed bool

	sendChan chan wsMessage // 所有写操作通过这个 channel

	reconnectCh  chan struct{}
	maxRetry     int
	stopCh       chan struct{}
	reconnecting bool

	failCount   int
	failLimit   int
	backoffBase time.Duration
}

// ------------------ 创建客户端 ------------------

func NewWSClient() *WSClient {
	c := &WSClient{
		maxRetry:    3,
		reconnectCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
		failLimit:   3,
		backoffBase: 1 * time.Second,
		sendChan:    make(chan wsMessage, 100),
	}
	go c.reconnectWorker()
	go c.writePump() // 启动写 goroutine
	return c
}

// ------------------ 启动连接 ------------------

func (c *WSClient) Start(url string) error {
	c.url = url
	if !IsRunning() {
		return fmt.Errorf("引擎未启动")
	}
	return c.doConnect()
}

// ------------------ 真正执行连接 ------------------

func (c *WSClient) doConnect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout:  5 * time.Second,
		EnableCompression: true,
	}

	var conn *websocket.Conn
	var err error

	for i := 1; i <= c.maxRetry; i++ {
		conn, _, err = dialer.Dial(c.url, nil)
		if err == nil {
			c.rw.Lock()
			c.conn = conn
			c.closed = false
			c.failCount = 0

			if c.stopCh == nil {
				c.stopCh = make(chan struct{})
			}

			c.rw.Unlock()

			log.Println("✅ 引擎连接成功")
			go c.heartbeat()
			return nil
		}
		time.Sleep(time.Duration(i*2) * time.Second)
	}
	return fmt.Errorf("引擎连接失败: %w", err)
}

// ================== 写 goroutine ==================

func (c *WSClient) writePump() {
	for msg := range c.sendChan {

		if !IsRunning() || !c.IsOnline() {
			select {
			case msg.errChan <- errors.New("引擎未运行或连接不在线，已丢弃"):
			default:
			}
			continue
		}

		c.rw.RLock()
		conn := c.conn
		closed := c.closed
		c.rw.RUnlock()

		if closed || conn == nil {
			msg.errChan <- errors.New("连接不存在")
			continue
		}

		if err := conn.WriteJSON(msg.req); err != nil {
			msg.errChan <- err
			c.triggerReconnect()
			continue
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			msg.errChan <- err
			c.triggerReconnect()
			continue
		}

		var resp Response
		if err := json.Unmarshal(data, &resp); err != nil {
			msg.errChan <- err
			continue
		}
		msg.respChan <- resp
	}
}

// ================== heartbeat ==================

func (c *WSClient) heartbeat() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pingReq := Request{Action: "ping"}
			respChan := make(chan Response, 1)
			errChan := make(chan error, 1)

			c.sendChan <- wsMessage{
				req:      pingReq,
				respChan: respChan,
				errChan:  errChan,
			}

			select {
			case <-respChan:
				c.rw.Lock()
				c.failCount = 0
				c.rw.Unlock()
			case <-errChan:
				c.rw.Lock()
				c.failCount++
				log.Printf("⚠️ 心跳失败 #%d", c.failCount)
				if c.failCount >= c.failLimit && !c.reconnecting {
					c.rw.Unlock()
					log.Println("⚠️ 引擎存活检测停止 ...")
					go c.triggerReconnect()
					return
				} else {
					c.rw.Unlock()
				}
			case <-c.stopCh:
				return
			}
		case <-c.stopCh:
			return
		}
	}
}

// ================== 重连控制 ==================

func (c *WSClient) triggerReconnect() {
	c.rw.Lock()
	defer c.rw.Unlock()
	if c.reconnecting || c.closed {
		return
	}
	c.reconnecting = true
	select {
	case c.reconnectCh <- struct{}{}:
	default:
	}
}

func (c *WSClient) reconnectWorker() {
	for range c.reconnectCh {
		log.Println("🔄 执行引擎重连...")
		c.CloseConn(false)

		backoff := c.backoffBase
		success := false
		for i := 0; i < c.maxRetry; i++ {
			if err := c.doConnect(); err != nil {
				if !IsRunning() {
					if !c.RestartLic() {
						err = errors.New("引擎停止运行")
					}
				}
				log.Printf("❌ 引擎重连第 %d 次失败: %v", i+1, err)
				time.Sleep(backoff)
				backoff *= 2
			} else {
				success = true
				break
			}
		}

		if !success {
			log.Println("❌ 重连失败，关闭连接")
			c.CloseConn(true)
		}

		c.rw.Lock()
		c.reconnecting = false
		c.failCount = 0
		c.rw.Unlock()
	}
}

// ================== 安全关闭 ==================

func (c *WSClient) CloseConn(fullClose bool) {
	c.rw.Lock()
	defer c.rw.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	if fullClose {
		c.closed = true
		select {
		case <-c.stopCh:
		default:
			close(c.stopCh)
		}
		c.stopCh = nil
	}
}

// ================== 连接状态 ==================

func (c *WSClient) IsOnline() bool {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.conn != nil && !c.closed && IsRunning()
}

// ================== 发送请求 ==================

func (c *WSClient) SendWS(req Request) (Response, error) {
	if !IsRunning() {
		if !c.RestartLic() {
			return Response{}, fmt.Errorf("引擎重启失败")
		}
		if !c.IsOnline() {
			if err := c.doConnect(); err != nil {
				return Response{}, fmt.Errorf("引擎未在线")
			}
		}
		return Response{}, fmt.Errorf("引擎未启动")
	}

	if !c.IsOnline() {
		if err := c.doConnect(); err != nil {
			return Response{}, fmt.Errorf("引擎未在线")
		}
	}

	for i := 0; i < c.maxRetry; i++ {
		respChan := make(chan Response, 1)
		errChan := make(chan error, 1)

		c.sendChan <- wsMessage{
			req:      req,
			respChan: respChan,
			errChan:  errChan,
		}

		select {
		case resp := <-respChan:
			return resp, nil
		case <-errChan:
			log.Printf("⚠️ 任务发送失败, 重试第 %d 次", i+1)
			time.Sleep(2 * time.Second)
		}
	}

	return Response{}, errors.New("发送失败，超过最大重试")
}

// ================== 引擎状态检测 ==================

func IsRunning() bool {
	cmd := exec.Command("bash", "-c", "ps -ef | grep '/license' | grep -v grep")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return checkRun()
	}
	return strings.Contains(string(output), "license")
}

func checkRun() bool {
	req, err := http.NewRequest("GET", "http://127.0.0.1:81/", nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return strings.Contains(string(body), "ok")
}

// ================== 重启引擎 ==================

func (c *WSClient) RestartLic() bool {
	log.Println("♻️ 正在重启引擎...")

	r := GetUrlData("http://127.0.0.1:82/licRestart")
	if strings.TrimSpace(r) == "" {
		log.Println("重启失败: 升级服务未启动")
		return false
	}
	if strings.TrimSpace(r) != "OK" {
		log.Println("重启失败: 升级服务返回错误")
		return false
	}

	err := c.Start("ws://127.0.0.1:81/ws")
	if err != nil {
		log.Println("引擎连接失败：", err)
		return false
	}

	res, err := c.SendWS(Request{Action: "getlic"})
	if err == nil {
		if err := json.Unmarshal(res.Data, &Lic); err == nil {
			log.Println("引擎初始化成功")
		} else {
			log.Println("授权信息解析错误:", err)
		}
	} else {
		log.Println("引擎初始化错误")
		return false
	}

	log.Println("✅  引擎已成功重启并重新连接")
	return true
}

func GetUrlData(url string, ua ...string) string {
	defaultUA := "Go-http-client/1.1"
	useUA := defaultUA

	if len(ua) > 0 && ua[0] != "" {
		useUA = ua[0]
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("User-Agent", useUA)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return string(body)
}
