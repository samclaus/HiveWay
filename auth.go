package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shamaton/msgpack/v2"
	"gorm.io/gorm"
)

const (
	RankNormal = iota
	RankAdmin
	RankRoot
)

// UserInfo represents user authentication information stored in the sqlite3 database.
// Salt and rounds are randomly generated per user when a new user is created.
// The password is NEVER stored in the database, only the hashed version.
type UserInfo struct {
	ID           string `gorm:"primaryKey" json:"id" msgpack:"id"`
	Username     string `gorm:"unique" json:"-" msgpack:"-"`
	Salt         []byte `gorm:"salt" json:"-" msgpack:"-"`
	Rounds       uint   `gorm:"rounds" json:"-" msgpack:"-"`
	PasswordHash []byte `gorm:"password_hash" json:"-" msgpack:"-"`
	Rank         uint   `gorm:"rank" msgpack:"rank"`
	Name         string `gorm:"name" msgpack:"name"`
	Email        string `gorm:"email" msgpack:"email,omitempty"`
}

// AuthRequest is a simple username/password combination used to authenticate
// a user. The client (JavaScript running in a web browser, iOS app, etc.) must
// send this information (encoded using MessagePack) as the very first message
// after opening a websocket. Failure to do so, or sending weird bytes that are
// not a correctly encoded MessagePack structure with the username/password,
// will result in the connection being terminated.
type AuthRequest struct {
	Protocol uint `msgpack:"protocol"`
	// Registration token. If provided, it means a new account should be created.
	// The token will be checked against the root registration token and
	// the database. If the token is not valid, the registration attempt will fail.
	RegToken string `msgpack:"registration_token"`
	Username string `msgpack:"username"`
	Password []byte `msgpack:"password"`
	Email    string `msgpack:"email"`
}

type ServerInfo struct {
}

type SessionInfo struct {
}

type LoginSuccessful struct {
	Protocol uint        `msgpack:"protocol"`
	Server   ServerInfo  `msgpack:"server"`
	User     UserInfo    `msgpack:"user"`
	Session  SessionInfo `msgpack:"session"`
}

const (
	// UserPasswordSaltSize is the number of random bytes to generate as a salt when a
	// new user is registered. The salt will be prepended to the user's password before
	// hashing it to make things more difficult for brute force attackers who try to use
	// precomputed hashes (rainbow tables) to quickly figure out passwords just by looking
	// at password hashes in the database.
	UserPasswordSaltSize = 12
	// ClientHandshakeTimeout is the duration the server is willing to wait to receive the
	// client's side of the websocket handshake before it gives up and closes the connection.
	ClientHandshakeTimeout = 5 * time.Second
)

// TODO: some or even most of these errors should just be encoded as MessagePack and prepended
// an error-indicator byte (nonzero byte) when the program starts so we are not wastefully
// encoding them over and over and over--could even be done at compile time, potentially.
var (
	// ErrOpaqueFailure should be sent to a client when we do not want to disclose what exactly
	// went wrong (for fear of exposing a server vulnerability) but do want to let the client
	// know that the connection mechanism is at least working and an application-level error
	// caused the operation to fail
	ErrOpaqueFailure = ErrorWithCode{"unspecified", "a system-level error occurred (please let an administrator know)", nil}
	// ErrHandshakeTimeout indicates that the client took too long to send its half of the authentication
	// handshake down the wire.
	ErrHandshakeTimeout = ErrorWithCode{
		Code:    "handshake-timeout",
		Message: fmt.Sprintf("did not receive client handshake within %s", ClientHandshakeTimeout.String()),
		Details: struct {
			TimeoutMilliseconds uint `msgpack:"timeout_milliseconds"`
		}{
			TimeoutMilliseconds: uint(ClientHandshakeTimeout.Milliseconds()),
		},
	}
	// ErrBadHandshakeMessagePack (TODO...)
	ErrBadHandshakeMessagePack = ErrorWithCode{"bad-handshake-encoding", "client handshake was not valid MessagePack or outer value was not a map", nil}
	// ErrBadHandshakeSchema (TODO...)
	// TODO: compute this on a case-by-case basis and say which fields were not included
	ErrBadHandshakeSchema = ErrorWithCode{"bad-handshake-schema", "client handshake did not contain required fields", nil}
	// ErrBadRegistrationToken means they DID provide a registration token, but it was either the
	// root token and a root account is already present, or there were no matching entries
	// in the table of admin-managed registration tokens.
	ErrBadRegistrationToken = ErrorWithCode{"bad-registration-token", "registration token is invalid", nil}
	// ErrUsernameTaken is used to tell the client that they cannot register with the provided
	// username because another account already exists with the same username. It is very
	// important that this error only be sent after the registration is validated using a
	// registration token or some other form of privilege because we do not want anyone on the
	// internet spamming the registration to guess-and-check find out usernames
	ErrUsernameTaken = ErrorWithCode{"username-taken", "username is taken", nil}
	// ErrWrongUsernameOrPassword indicates that the username, password, or both did not correspond
	// to an existing account. The error is intentionally vague because we do not want anyone
	// (including bots) on the internet just spamming the authentication API to figure out existing
	// account usernames.
	ErrWrongUsernameOrPassword = ErrorWithCode{"bad-username-password", "no account found with that username/password combination", nil}
)

func hashAndScrubPassword(pwd, salt []byte, rounds uint) [32]byte {
	// Create a new buffer with the salt prepended to the password
	saltedPassword := append(salt, pwd...)

	// Now scrub the original password buffer--idea is to clean things from memory as quick
	// as possible in case an attacker is watching
	scrub(pwd)

	// Now create a 32-byte SHA256 hash of the salt+password combo
	passwordHash := sha256.Sum256(saltedPassword)

	// Going forward we will just continually hash in-place using the 32-byte array, so we
	// can immediately scrub the salt+password combo from memory since it still contains the
	// password!
	scrub(saltedPassword)

	// Start i at 1 because we already did 1 round of hashing to create the 32-byte array above
	for i := uint(1); i < rounds; i++ {
		passwordHash = sha256.Sum256(passwordHash[:])
	}

	return passwordHash
}

// writeHandshakeErrorOrLog encodes a standard error reply struct using MessagePack and sends it
// down the wire with a prefixed byte containing the value '1' (just has to be nonzero) to tell
// the client that the handshake failed and the rest of the message bytes are an error
func writeHandshakeErrorOrLog(ws *websocket.Conn, errRes ErrorWithCode) {
	msg, err := msgpack.Marshal(errRes)
	if err != nil {
		log.Printf("Failed to encode handshake error [%v] as MessagePack: %v", errRes, err)
		return
	}

	if err = ws.WriteMessage(websocket.BinaryMessage, append([]byte{1}, msg...)); err != nil {
		log.Printf("Failed to write handshake error [%v] to websocket: %v", errRes, err)
		return
	}
}

// ServeHTTP implements the http.Handler interface for Server. The only HTTP route provided
// is '/connect', which immediately upgrades request connections to websockets and authenticates
// them as either a new user (registering) or existing user (logging in).
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// We COULD just ignore the URL and try to upgrade to a websocket no matter what, but I want
	// to establish a strict API going forward and there likely WILL be HTTP-based APIs added in
	// the future
	if r.URL.Path != "/connect" {
		http.Error(
			w,
			"Resource not found - use the '/connect' URL to register/login and communicate over a WebSocket.",
			http.StatusNotFound,
		)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// The .Upgrade() call promises to send an error reply to the browser if upgrading fails so
		// we can just log the error here and return.
		log.Printf("Failed to upgrade connection to WebSocket: %v", err)
		return
	}

	defer ws.Close()

	// We expect the client to send an authentication message within 5s
	ws.SetReadDeadline(time.Now().Add(ClientHandshakeTimeout))
	_, authMsg, err := ws.ReadMessage()
	ws.SetReadDeadline(time.Time{}) // reset to zero-value which means no timeout
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			writeHandshakeErrorOrLog(ws, ErrHandshakeTimeout)
		} else {
			log.Printf("Failed to read login request: %v", err)
			writeHandshakeErrorOrLog(ws, ErrOpaqueFailure)
		}
		return
	}

	var auth AuthRequest
	err = msgpack.Unmarshal(authMsg, &auth)
	if err != nil {
		// IMPORTANT: do not log the raw request (as, say, hex) because it likely does
		// contain the password or other sensitive information that we don't want sitting
		// around in log files!
		// Also, if the client is incapable of encoding a handshake as proper MessagePack,
		// it probably won't understand the MessagePack error response we send it--but we
		// will send one just in case the decoding code is correct on the client-side and
		// the encoding code is the only part that is broken.
		writeHandshakeErrorOrLog(ws, ErrBadHandshakeMessagePack)
		return
	}

	// This will be populated differently depending on whether a new user is registering
	// or an existing user is logging in
	var user UserInfo

	if auth.RegToken != "" {
		// TODO: validate the username (length, allowed characters, etc.)
		// TODO: probably ensure a certain password length (unsure right now)
		// TODO: user/registration_token tables should all be updated in a single transaction
		// to avoid issues like creating a user but the registration token is still there, or
		// able to be used multiple times if people register at exactly the same time

		var rank uint
		var name string
		deleteRegToken := false

		if auth.RegToken == s.RootRegToken {
			rank = RankRoot
			name = "Root"

			var adminCount int64
			if err := s.Database.Model(&UserInfo{}).Where("rank = 1").Count(&adminCount).Error; err != nil {
				log.Printf("Failed to query number of existing admins: %v", err)
				writeHandshakeErrorOrLog(ws, ErrOpaqueFailure)
				return
			}

			if adminCount > 0 {
				writeHandshakeErrorOrLog(ws, ErrBadRegistrationToken)
				return
			}
		} else {
			var token RegistrationTokenInfo

			if err = s.Database.Take(&token, "id = ?", auth.RegToken).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					writeHandshakeErrorOrLog(ws, ErrBadRegistrationToken)
				} else {
					log.Printf("Failed to query registration token [%s]: %v", auth.RegToken, err)
					writeHandshakeErrorOrLog(ws, ErrOpaqueFailure)
				}
				return
			}

			rank = token.Rank
			name = token.Name
			deleteRegToken = true
		}

		id, err := uuid.NewRandom()
		if err != nil {
			log.Printf("Failed to generate UUID for new user %q: %v", auth.Username, err)
			writeHandshakeErrorOrLog(ws, ErrOpaqueFailure)
			return
		}

		roundsOfHashing := uint(20_000) // TODO: make this random

		var pwdSalt [UserPasswordSaltSize]byte
		if _, err = rand.Read(pwdSalt[:]); err != nil {
			log.Printf("Failed to generate password salt for new user %q: %v", auth.Username, err)
			writeHandshakeErrorOrLog(ws, ErrOpaqueFailure)
			return
		}

		pwdHash := hashAndScrubPassword(auth.Password, pwdSalt[:], roundsOfHashing)
		user = UserInfo{
			ID:           id.String(),
			Username:     auth.Username,
			Salt:         pwdSalt[:],
			Rounds:       roundsOfHashing,
			PasswordHash: pwdHash[:],
			Rank:         rank,
			Name:         name,
			Email:        auth.Email, // TODO: enforce that they specified a well-formed email address
		}

		if err := s.Database.Create(user).Error; err != nil {
			// TODO: it would be nice if we had a standardized API for checking unique constraint
			// database errors--GORM only includes a proper way to detect "record not found" errors
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				writeHandshakeErrorOrLog(ws, ErrUsernameTaken)
			} else {
				log.Printf("Failed to insert new user %q into database: %v", auth.Username, err)
				writeHandshakeErrorOrLog(ws, ErrOpaqueFailure)
			}
			return
		}

		if deleteRegToken {
			if err := s.Database.Delete(&RegistrationTokenInfo{}, "id = ?", auth.RegToken).Error; err != nil {
				log.Printf("Failed to delete token [%s] after successful registration: %v", auth.RegToken, err)
			}
		}
	} else {
		if err = s.Database.Take(&user, &UserInfo{Username: auth.Username}).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				writeHandshakeErrorOrLog(ws, ErrWrongUsernameOrPassword)
			} else {
				log.Printf("Failed to lookup %q from database: %v", auth.Username, err)
				writeHandshakeErrorOrLog(ws, ErrOpaqueFailure)
			}
			scrub(auth.Password)
			return
		}

		if pwdHash := hashAndScrubPassword(auth.Password, user.Salt, user.Rounds); !bytes.Equal(pwdHash[:], user.PasswordHash) {
			writeHandshakeErrorOrLog(ws, ErrWrongUsernameOrPassword)
			return
		}
	}

	successReply, err := msgpack.Marshal(LoginSuccessful{
		Protocol: 0,
		Server:   ServerInfo{},
		User:     user,
		Session:  SessionInfo{},
	})
	if err != nil {
		// If the MessagePack library fails to encode the response, something is seriously wrong
		log.Printf("Failed to encode authentication success reply for %q: %v", auth.Username, err)
		// TODO: we need to MessagePack-encode the opaque failure error when the program starts and
		// kill it (panic) if encoding fails
		writeHandshakeErrorOrLog(ws, ErrOpaqueFailure)
		return
	}

	if err = ws.WriteMessage(websocket.BinaryMessage, append([]byte{0}, successReply...)); err != nil {
		// If we fail to send the success reply on the websocket, something is very wrong and we
		// probably won't be able to send an error reply either, so just log the error and kill it.
		log.Printf("Failed to send authentication success reply for %q: %v", auth.Username, err)
		return
	}

	// SUCCESS! Once the serve function eventually returns, the websocket will be automatically cleaned
	// up thanks to the 'defer' statement immediately after the websocket initialization code above
	if err = s.ServeAuthenticatedConn(ws, user); err != nil {
		log.Printf("Unexpectedly stopped serving connection for %q: %v", auth.Username, err)
	}
}
