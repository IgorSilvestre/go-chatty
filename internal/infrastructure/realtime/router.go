package realtime

import (
	"sync"
)

// Router coordinates websocket sessions and logical rooms (conversations).
// It keeps one active Connection per user while allowing efficient fan-out
// to all members subscribed to a conversation.
type Router struct {
	mu           sync.RWMutex
	sessions     map[string]*Connection            // sessionID -> connection
	userSessions map[string]string                 // userID -> sessionID
	rooms        map[string]map[string]*Connection // conversationID -> sessionID -> connection
	sessionRooms map[string]map[string]struct{}    // sessionID -> set of conversationIDs
}

// NewRouter constructs an initialized Router.
func NewRouter() *Router {
	return &Router{
		sessions:     make(map[string]*Connection),
		userSessions: make(map[string]string),
		rooms:        make(map[string]map[string]*Connection),
		sessionRooms: make(map[string]map[string]struct{}),
	}
}

// Attach registers a connection for the given user. If a previous session exists,
// it is removed and closed after the swap to enforce one active socket per user.
func (r *Router) Attach(conn *Connection) {
	var previous *Connection

	r.mu.Lock()
	if existingID, ok := r.userSessions[conn.UserID]; ok {
		if existing := r.sessions[existingID]; existing != nil {
			previous = existing
			r.detachLocked(existingID)
		}
	}

	r.sessions[conn.ID] = conn
	r.userSessions[conn.UserID] = conn.ID
	r.sessionRooms[conn.ID] = make(map[string]struct{})
	r.mu.Unlock()

	conn.Start()

	if previous != nil {
		previous.Close(4001, "session replaced")
	}
}

// Detach removes a connection if it is still tracked.
func (r *Router) Detach(conn *Connection) {
	r.mu.Lock()
	r.detachLocked(conn.ID)
	r.mu.Unlock()
}

// Join adds the connection to the conversation room.
func (r *Router) Join(conversationID string, conn *Connection) {
	r.mu.Lock()
	if _, ok := r.sessions[conn.ID]; !ok {
		r.mu.Unlock()
		return
	}

	room := r.rooms[conversationID]
	if room == nil {
		room = make(map[string]*Connection)
		r.rooms[conversationID] = room
	}
	room[conn.ID] = conn

	memberships := r.sessionRooms[conn.ID]
	if memberships == nil {
		memberships = make(map[string]struct{})
		r.sessionRooms[conn.ID] = memberships
	}
	memberships[conversationID] = struct{}{}
	r.mu.Unlock()
}

// Leave removes the connection from the conversation room.
func (r *Router) Leave(conversationID string, conn *Connection) {
	r.mu.Lock()
	r.leaveLocked(conversationID, conn.ID)
	r.mu.Unlock()
}

// Broadcast writes payload to all members in the conversation.
// excludeUserID, when non-empty, prevents delivering to that user.
func (r *Router) Broadcast(conversationID string, payload []byte, excludeUserID string) int {
	r.mu.RLock()
	room := r.rooms[conversationID]
	if len(room) == 0 {
		r.mu.RUnlock()
		return 0
	}

	delivered := 0
	for _, conn := range room {
		if excludeUserID != "" && conn.UserID == excludeUserID {
			continue
		}
		if err := conn.Send(payload); err == nil {
			delivered++
		}
	}
	r.mu.RUnlock()
	return delivered
}

// NotifyUser delivers payload to the current connection of the given user.
func (r *Router) NotifyUser(userID string, payload []byte) bool {
	r.mu.RLock()
	sessionID, ok := r.userSessions[userID]
	if !ok {
		r.mu.RUnlock()
		return false
	}
	conn := r.sessions[sessionID]
	r.mu.RUnlock()
	if conn == nil {
		return false
	}
	return conn.Send(payload) == nil
}

// Close terminates all tracked connections and clears router state.
func (r *Router) Close() {
	r.mu.Lock()
	sessions := make([]*Connection, 0, len(r.sessions))
	for _, conn := range r.sessions {
		sessions = append(sessions, conn)
	}
	r.sessions = make(map[string]*Connection)
	r.userSessions = make(map[string]string)
	r.rooms = make(map[string]map[string]*Connection)
	r.sessionRooms = make(map[string]map[string]struct{})
	r.mu.Unlock()

	for _, conn := range sessions {
		conn.Close(1001, "router shutdown")
	}
}

func (r *Router) detachLocked(sessionID string) {
	conn, ok := r.sessions[sessionID]
	if !ok {
		return
	}
	delete(r.sessions, sessionID)

	if current, ok := r.userSessions[conn.UserID]; ok && current == sessionID {
		delete(r.userSessions, conn.UserID)
	}

	for roomID := range r.sessionRooms[sessionID] {
		r.leaveLocked(roomID, sessionID)
	}
	delete(r.sessionRooms, sessionID)
}

func (r *Router) leaveLocked(conversationID string, sessionID string) {
	if sessionID == "" {
		return
	}
	room := r.rooms[conversationID]
	if room == nil {
		return
	}
	delete(room, sessionID)
	if len(room) == 0 {
		delete(r.rooms, conversationID)
	}
	if memberships, ok := r.sessionRooms[sessionID]; ok {
		delete(memberships, conversationID)
		if len(memberships) == 0 {
			delete(r.sessionRooms, sessionID)
		}
	}
}
