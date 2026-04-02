package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	SendBin  chan []byte
	RoomID   string
	UserID   string
	Username string
}

// ReadPump читает сообщения от клиента
func (c *Client) ReadPump() {
	defer func() {
		room := c.Hub.GetOrCreateRoom(c.RoomID)
		if room != nil {
			room.Unregister <- c
		}
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		mt, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Read error from %s: %v", c.UserID, err)
			break
		}
		if mt == websocket.BinaryMessage {
			log.Printf("Binary update from %s, len=%d", c.UserID, len(message))
			c.Hub.GetOrCreateRoom(c.RoomID).BroadcastBin <- message
		} else if mt == websocket.TextMessage {
			var cmd map[string]interface{}
			if err := json.Unmarshal(message, &cmd); err == nil {
				cmdType, _ := cmd["type"].(string)
				switch cmdType {
				case "join":
					if payload, ok := cmd["payload"].(map[string]interface{}); ok {
						if username, ok := payload["username"].(string); ok {
							c.Username = username
						}
					}
					c.broadcastParticipants()
				case "cursor":
					c.broadcastCursor(cmd)
				// При получении save_text, обновляем и язык
				case "save_text":
					log.Printf("Received save_text from %s", c.UserID)
					if payload, ok := cmd["payload"].(map[string]interface{}); ok {
						if content, ok := payload["content"].(string); ok {
							room := c.Hub.GetOrCreateRoom(c.RoomID)
							if room != nil {
								// Если есть язык в payload
								if lang, ok := payload["language"].(string); ok {
									room.SetLanguage(lang)
								}
								room.SaveText(content, c.UserID, c.Username)
							}
						}
					}
				case "run_code":
					log.Printf("Received run_code from %s", c.UserID)
					room := c.Hub.GetOrCreateRoom(c.RoomID)
					if room == nil {
						c.SendError("room not found")
						continue
					}
					output, err := room.RunCode()
					result := map[string]interface{}{
						"type":   "code_output",
						"output": output,
						"error":  err != nil,
					}
					if err != nil {
						result["error_msg"] = err.Error()
					}
					data, _ := json.Marshal(result)
					room.BroadcastText <- data
				case "terminal_exec":
					log.Printf("Received terminal_exec from %s", c.UserID)
					room := c.Hub.GetOrCreateRoom(c.RoomID)
					if room == nil {
						c.SendError("room not found")
						continue
					}
					var command string
					if payload, ok := cmd["payload"].(map[string]interface{}); ok {
						if cmdStr, ok := payload["command"].(string); ok {
							command = cmdStr
						}
					}
					if command == "" {
						c.SendError("empty command")
						continue
					}

					// Для cd и других shell команд используем /bin/sh
					var finalCommand string
					if command == "cd" || command == "pwd" || command == "ls" || command == "whoami" {
						finalCommand = command
					} else if len(command) > 2 && command[:2] == "cd" {
						// Поддержка cd с аргументами
						finalCommand = command
					} else {
						finalCommand = command
					}

					output, err := room.Exec(finalCommand)
					result := map[string]interface{}{
						"type":    "terminal_output",
						"command": command,
						"output":  output,
						"error":   err != nil,
					}
					if err != nil {
						result["error_msg"] = err.Error()
					}
					data, _ := json.Marshal(result)
					room.BroadcastText <- data
				default:
					c.Hub.GetOrCreateRoom(c.RoomID).BroadcastText <- message
				}
			} else {
				c.Hub.GetOrCreateRoom(c.RoomID).BroadcastText <- message
			}
		}
	}
}

func (c *Client) SendError(msg string) {
	errMsg := map[string]interface{}{
		"type":    "error",
		"message": msg,
	}
	data, _ := json.Marshal(errMsg)
	c.Send <- data
}

// broadcastParticipants отправляет список участников всем в комнате
func (c *Client) broadcastParticipants() {
	room := c.Hub.GetOrCreateRoom(c.RoomID)
	participants := make([]map[string]interface{}, 0, len(room.Clients))
	for cl := range room.Clients {
		participants = append(participants, map[string]interface{}{
			"user_id":  cl.UserID,
			"username": cl.Username,
		})
	}
	msg := map[string]interface{}{
		"type":    "participants",
		"payload": participants,
	}
	data, _ := json.Marshal(msg)
	room.BroadcastText <- data
}

// broadcastCursor рассылает обновление позиции курсора всем, кроме отправителя
func (c *Client) broadcastCursor(cmd map[string]interface{}) {
	room := c.Hub.GetOrCreateRoom(c.RoomID)
	msg := map[string]interface{}{
		"type":     "cursor_update",
		"payload":  cmd["payload"],
		"user_id":  c.UserID,
		"username": c.Username,
	}
	data, _ := json.Marshal(msg)
	for client := range room.Clients {
		if client != c {
			select {
			case client.Send <- data:
			default:
			}
		}
	}
}

// WritePump отправляет сообщения клиенту
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write text error: %v", err)
				return
			}
		case binMsg, ok := <-c.SendBin:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.BinaryMessage, binMsg); err != nil {
				log.Printf("Write binary error: %v", err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
