package models

import (
	"encoding/json"
	"time"
)

type Question struct {
	Letter   string `json:"letter"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type GameState struct {
	SessionID     string            `json:"sessionId"`
	Language      string            `json:"language"`
	Questions     []Question        `json:"-"`
	CurrentIndex  int               `json:"currentIndex"`
	CorrectCount  int               `json:"correctCount"`
	WrongCount    int               `json:"wrongCount"`
	PassedCount   int               `json:"passedCount"`
	LettersState  map[string]string `json:"lettersState"` // "green", "red", "yellow", ""
	TimeRemaining int               `json:"timeRemaining"` // in seconds
	IsGameOver    bool              `json:"isGameOver"`
	StartTime     time.Time         `json:"-"`
	PendingPasses []int             `json:"-"` // Indices of questions marked as yellow
	IsRevisiting  bool              `json:"-"`
}

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type InitPayload struct {
	Language  string `json:"language"`
	SessionID string `json:"sessionId"`
}

type AnswerPayload struct {
	Answer string `json:"answer"`
}

type QuestionPayload struct {
	Letter        string            `json:"letter"`
	Question      string            `json:"question"`
	Index         int               `json:"index"`
	LettersState  map[string]string `json:"lettersState"`
	TimeRemaining int               `json:"timeRemaining"`
}

type FeedbackPayload struct {
	Letter  string `json:"letter"`
	Status  string `json:"status"` // green, red, yellow
	Correct bool   `json:"correct"`
}

type GameOverPayload struct {
	CorrectCount int `json:"correctCount"`
	WrongCount   int `json:"wrongCount"`
	PassedCount  int `json:"passedCount"`
}

const (
	TypeInit      = "INIT"
	TypeQuestion  = "QUESTION"
	TypeAnswer    = "ANSWER"
	TypePass      = "PASS"
	TypeFeedback  = "FEEDBACK"
	TypeTimerSync = "TIMER_SYNC"
	TypeGameOver  = "GAME_OVER"
	TypeError     = "ERROR"
)
