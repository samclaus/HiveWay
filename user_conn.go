package main

import (
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

func writeResponseOrLog(c *websocket.Conn, rid uint32, resType byte, res any) {
	var payload []byte
	var err error

	if res != nil {
		if payload, err = msgpack.Marshal(res); err != nil {
			log.Printf(
				"Failed to encode response [ID:%d, Error:%t, Payload:%T]: %v",
				rid, resType != ReplyTypeSuccess, res, err,
			)
			return
		}
	}

	header := make([]byte, 6, 6+len(payload)) // allocate enough capacity for payload up front
	header[0] = ProtocolRequestReply
	binary.BigEndian.PutUint32(header[1:], rid)
	header[5] = resType

	if err = c.WriteMessage(websocket.BinaryMessage, append(header, payload...)); err != nil {
		log.Printf(
			"Failed to write response [ID:%d, Error:%t]: %v",
			rid, resType != ReplyTypeSuccess, err,
		)
	}
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

		if len(msg) < 7 {
			// TODO: only log in debug mode, or maybe close connection if we receive a short message
			log.Printf("Received short (%d bytes) message from %q", len(msg), u.Username)
			continue
		}

		switch msg[0] {
		case ProtocolRequestReply:
			rid := binary.BigEndian.Uint32(msg[1:])

			if msg[5] != 0xd9 {
				writeResponseOrLog(c, rid, ReplyTypeError, ErrorWithCode{
					"non-str-8-request-type",
					"server currently only supports MessagePack str-8 encoding for request types",
					nil,
				})
				continue
			}

			rtypeLen := msg[6]
			rtype := string(msg[7 : 7+rtypeLen])
			payload := msg[7+rtypeLen:]
			handler, knownType := s.RequestHandlers[rtype]

			if !knownType {
				writeResponseOrLog(c, rid, ReplyTypeError, ErrorWithCode{
					"unknown-request-type",
					fmt.Sprintf("%q is not a recognized request type", rtype),
					rtype,
				})
				continue
			}

			res, err := handler(s, &u, payload)
			if err != nil {
				writeResponseOrLog(c, rid, ReplyTypeError, err)
			} else {
				writeResponseOrLog(c, rid, ReplyTypeSuccess, res)
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
