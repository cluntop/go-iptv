package dao

import (
	"encoding/json"
)

var WS = NewWSClient()
var Lic struct {
	Status int `json:"status"`
}

type Request struct {
	Action string      `json:"a"`
	Data   interface{} `json:"d"`
}

type Response struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type WSClient struct {
	closed bool
}

func NewWSClient() *WSClient {
	return &WSClient{
		closed: true,
	}
}

func (c *WSClient) Start(url string) error {
	return nil
}

func (c *WSClient) SendWS(req Request) (Response, error) {
	return Response{Code: 1}, nil
}

func (c *WSClient) IsOnline() bool {
	return false
}

func (c *WSClient) RestartLic() bool {
	return true
}

func (c *WSClient) CloseConn(fullClose bool) {
	c.closed = true
}

func IsRunning() bool {
	return false
}
