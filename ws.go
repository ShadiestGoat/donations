package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 0,
	ReadBufferSize:   0,
	WriteBufferSize:  0,
	WriteBufferPool:  nil,
	Subprotocols:     []string{},
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
	},
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	if NoPermHTTP(w, r, PERM_LIVE_NOTIFICATION) {
		return
	}
	app := Apps[r.Header.Get("Authorization")]

	if _, ok := WSMgr.Connections[app.Token]; ok {
		RespondErr(w, ErrDoubleAccess)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		Respond(w, 501, []byte(`{"error": "`+err.Error()+`"}`))
		return
	}

	WSMgr.Add(app.Token, conn)
}

type EventType int

const (
	ET_NONE EventType = iota
	ET_NEW_DON
	ET_NEW_FUND
	ET_PING
)

type WSR_NewDon struct {
	*Donation
}

func (e WSR_NewDon) WSEvent() WSEvent {
	b, _ := json.Marshal(e)
	return WSEvent{
		Type: ET_NEW_DON,
		Body: b,
	}
}

type WSR_NewFund struct {
	*Fund
}

func (e WSR_NewFund) WSEvent() WSEvent {
	b, _ := json.Marshal(e)
	return WSEvent{
		Type: ET_NEW_FUND,
		Body: b,
	}
}

type WSEvent struct {
	Type EventType       `json:"event"`
	Body json.RawMessage `json:"body"`
	Perm Permission      `json:"-"`
}

type WSMgrT struct {
	Lock *sync.RWMutex
	// token : conn
	Connections map[string]*websocket.Conn
}

var WSMgr = &WSMgrT{
	Lock:        &sync.RWMutex{},
	Connections: map[string]*websocket.Conn{},
}

func (mgr *WSMgrT) Add(token string, conn *websocket.Conn) {
	mgr.Lock.Lock()
	defer mgr.Lock.Unlock()

	mgr.Connections[token] = conn
}

func (mgr *WSMgrT) Remove(token string, reason string) {
	mgr.Lock.Lock()
	defer mgr.Lock.Unlock()
	logger.Logf(LL_DEBUG, "Closing WS Connection for '%v': %v", Apps[token].Name, reason)
	conn := mgr.Connections[token]
	conn.WriteControl(websocket.CloseMessage, []byte{}, time.Time{})
	conn.Close()
	delete(mgr.Connections, token)
}

func (mgr *WSMgrT) SendEvent(e WSEvent) {
	enc, _ := json.Marshal(e)
	prepared, _ := websocket.NewPreparedMessage(websocket.TextMessage, enc)

	mgr.Lock.Lock()
	defer mgr.Lock.Unlock()

	for token, c := range mgr.Connections {
		if HasPerm(token, e.Perm) {
			c.WritePreparedMessage(prepared)
		}
	}
}

func (mgr *WSMgrT) PingLoop() {
	for {
		time.Sleep(30 * time.Second)
		mgr.Ping()
	}
}

var (
	// Using a custom ping allows browsers to connect
	WS_PING *websocket.PreparedMessage
)

func init() {
	b, _ := json.Marshal(WSEvent{
		Type: ET_PING,
	})
	WS_PING, _ = websocket.NewPreparedMessage(1, b)
}

func (mgr *WSMgrT) Ping() {
	mgr.Lock.Lock()
	defer mgr.Lock.Unlock()

	wg := &sync.WaitGroup{}

	for id, c := range mgr.Connections {
		wg.Add(1)

		go func(id string, c *websocket.Conn) {
			c.WritePreparedMessage(WS_PING)		
			c.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, p, err := c.ReadMessage()

			if err != nil || len(p) == 0 || p[0] != 'P' {
				go mgr.Remove(id, "Ping fail!")
			} else {
				c.SetReadDeadline(time.Time{})
			}
			wg.Done()
		}(id, c)
	}
	wg.Wait()
}

func init() {
	go WSMgr.PingLoop()
}
