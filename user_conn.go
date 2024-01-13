package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	ProtocolRequestReply = iota
	ProtocolStream
)

const (
	ReplyTypeSuccess = iota
	ReplyTypeError
)

// UserConn is an authenticated websocket connection.
type UserConn struct {
	UserInfo
	*websocket.Conn
}

type DebugReplyPayload struct {
	UserID          string `msgpack:"userID"`
	Username        string `msgpack:"username"`
	ReqID           uint32 `msgpack:"reqID"`
	ReqType         string `msgpack:"reqType"`
	PayloadAsString string `msgpack:"payloadAsString"`
}

// Serve blocks, repeatedly reading and handling individual requests asynchronously until reading
// a message from the websocket fails. This function will return nil if the websocket was closed
// normally. To be clear, any error returned from this function will originate from a failed read,
// and will be from the websocket library, NOT a wrapper error.
func (s *Server) ServeAuthenticatedConn(c *websocket.Conn, u UserInfo) error {
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			// Do not treat a normal close as an error
			if websocket.IsCloseError(err, 1000) {
				return nil
			}
			return err // Do NOT wrap the error, since this function ONLY returns read errors
		}

		if len(msg) < 6 {
			// TODO: only log in debug mode, or maybe close connection if we receive a short message
			log.Printf("Received short (%d bytes) message from %q", len(msg), u.Username)
			continue
		}

		switch msg[0] {
		case ProtocolRequestReply:
			reqID := binary.BigEndian.Uint32(msg[1:])
			decoder := msgpack.NewDecoder(bytes.NewReader(msg[5:]))
			reqType, err := decoder.DecodeString()

			if err != nil {
				log.Printf(
					"Received message from %q with badly encoded (MessagePack) request type string: %v",
					u.Username,
					err,
				)
			}

			reqPayloadAsString, err := decoder.DecodeString()
			if err != nil {
				reqPayloadAsString = fmt.Sprintf("(payload was not MessagePack string: %v)", err)
			}

			header := make([]byte, 6)
			header[0] = ProtocolRequestReply
			binary.BigEndian.PutUint32(header[1:], reqID)
			header[5] = ReplyTypeSuccess

			var replyStruct interface{}

			if reqType == "user:list" {
				var users []UserInfo
				s.Database.Find(&users)
				replyStruct = users
			} else {
				replyStruct = DebugReplyPayload{
					UserID:          u.ID,
					Username:        u.Username,
					ReqID:           reqID,
					ReqType:         reqType,
					PayloadAsString: reqPayloadAsString,
				}
			}

			reply, err := msgpack.Marshal(replyStruct)
			if err != nil {
				log.Printf("Failed to encode debug reply message: %v", err)
			}

			if err = c.WriteMessage(websocket.BinaryMessage, append(header, reply...)); err != nil {
				log.Printf("Failed to send reply on websocket: %v", err)
			}

		case ProtocolStream:
			log.Printf("Received stream message from %q", u.Username)

		default:
			log.Printf(
				"Received invalid message protocol byte from %q (expected 0 or 1, got %d)",
				u.Username,
				msg[0],
			)
		}
	}
}
