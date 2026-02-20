package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"gopherword/internal/game"
	"gopherword/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	gm *game.GameManager
}

type SafeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (sc *SafeConn) WriteJSON(v interface{}) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteJSON(v)
}

func main() {
	gm, err := game.NewGameManager("data/questions.json")
	if err != nil {
		log.Fatal(err)
	}

	server := &Server{gm: gm}

	// Middleware to handle session cookie
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if _, err := r.Cookie("session_id"); err != nil {
				sessionID := uuid.New().String()
				http.SetCookie(w, &http.Cookie{
					Name:     "session_id",
					Value:    sessionID,
					Path:     "/",
					Expires:  time.Now().Add(24 * time.Hour),
					HttpOnly: true,
				})
			}
			http.ServeFile(w, r, "./static/index.html")
			return
		}
		http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
	})

	http.HandleFunc("/ws", server.handleWebSocket)

	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	sconn := &SafeConn{conn: conn}

	var sessionID string
	cookie, err := r.Cookie("session_id")
	if err != nil {
		sessionID = uuid.New().String()
	} else {
		sessionID = cookie.Value
	}

	stopTicker := make(chan bool, 1)
	go s.timerTicker(sconn, sessionID, stopTicker)
	defer func() {
		select {
		case stopTicker <- true:
		default:
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Println("unmarshal:", err)
			continue
		}

		s.processMessage(sconn, sessionID, wsMsg)
	}
}

func (s *Server) timerTicker(sconn *SafeConn, sessionID string, stop chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			state, ok := s.gm.GetState(sessionID)
			if ok && !state.IsGameOver {
				payload, _ := json.Marshal(map[string]int{"timeRemaining": state.TimeRemaining})
				sconn.WriteJSON(models.WSMessage{
					Type:    models.TypeTimerSync,
					Payload: payload,
				})
			} else if ok && state.IsGameOver {
				s.sendState(sconn, state)
				return
			}
		case <-stop:
			return
		}
	}
}

func (s *Server) processMessage(sconn *SafeConn, sessionID string, msg models.WSMessage) {
	switch msg.Type {
	case models.TypeInit:
		var payload models.InitPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Println("unmarshal payload:", err)
			return
		}

		id := sessionID
		if payload.SessionID != "" {
			id = payload.SessionID
		}

		state, _ := s.gm.GetOrCreateSession(id, payload.Language)
		s.sendState(sconn, state)

	case models.TypeAnswer:
		var payload models.AnswerPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Println("unmarshal payload:", err)
			return
		}

		state, feedback, ok := s.gm.SubmitAnswer(sessionID, payload.Answer)
		if ok {
			fbData, _ := json.Marshal(feedback)
			sconn.WriteJSON(models.WSMessage{Type: models.TypeFeedback, Payload: fbData})
			s.sendState(sconn, state)
		}

	case models.TypePass:
		state, feedback, ok := s.gm.Pass(sessionID)
		if ok {
			fbData, _ := json.Marshal(feedback)
			sconn.WriteJSON(models.WSMessage{Type: models.TypeFeedback, Payload: fbData})
			s.sendState(sconn, state)
		}

	default:
		log.Println("Unknown message type:", msg.Type)
	}
}

func (s *Server) sendState(sconn *SafeConn, state *models.GameState) {
	if state.IsGameOver {
		payload, _ := json.Marshal(models.GameOverPayload{
			CorrectCount: state.CorrectCount,
			WrongCount:   state.WrongCount,
			PassedCount:  len(state.PendingPasses),
		})
		sconn.WriteJSON(models.WSMessage{
			Type:    models.TypeGameOver,
			Payload: payload,
		})
		return
	}

	currentQ := state.Questions[state.CurrentIndex]
	payload, _ := json.Marshal(models.QuestionPayload{
		Letter:        currentQ.Letter,
		Question:      currentQ.Question,
		Index:         state.CurrentIndex,
		LettersState:  state.LettersState,
		TimeRemaining: state.TimeRemaining,
	})
	sconn.WriteJSON(models.WSMessage{
		Type:    models.TypeQuestion,
		Payload: payload,
	})
}
