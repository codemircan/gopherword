package game

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"gopherword/internal/models"
)

const GameDuration = 300 // 5 minutes

type GameManager struct {
	Sessions  map[string]*models.GameState
	Questions map[string][]models.Question
	mu        sync.RWMutex
}

func NewGameManager(questionsPath string) (*GameManager, error) {
	data, err := os.ReadFile(questionsPath)
	if err != nil {
		return nil, err
	}

	var questions map[string][]models.Question
	if err := json.Unmarshal(data, &questions); err != nil {
		return nil, err
	}

	return &GameManager{
		Sessions:  make(map[string]*models.GameState),
		Questions: questions,
	}, nil
}

func (gm *GameManager) GetOrCreateSession(sessionID, lang string) (*models.GameState, bool) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	state, exists := gm.Sessions[sessionID]
	if exists {
		// If language changed, we might want to reset, but the requirement is "keep game state"
		// If session exists, we return it.
		return state, false
	}

	// Create new session
	questions, ok := gm.Questions[lang]
	if !ok {
		// Default to English if language not found
		questions = gm.Questions["en"]
		lang = "en"
	}

	newState := &models.GameState{
		SessionID:     sessionID,
		Language:      lang,
		Questions:     questions,
		CurrentIndex:  0,
		LettersState:  make(map[string]string),
		TimeRemaining: GameDuration,
		StartTime:     time.Now(),
		IsGameOver:    false,
	}

	// Initialize letters state
	for _, q := range questions {
		newState.LettersState[q.Letter] = ""
	}

	gm.Sessions[sessionID] = newState
	return newState, true
}

func (gm *GameManager) SubmitAnswer(sessionID, answer string) (*models.GameState, models.FeedbackPayload, bool) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	state, ok := gm.Sessions[sessionID]
	if !ok || state.IsGameOver {
		return nil, models.FeedbackPayload{}, false
	}

	gm.updateTimer(state)
	if state.IsGameOver {
		return state, models.FeedbackPayload{}, true
	}

	currentQ := state.Questions[state.CurrentIndex]
	isCorrect := strings.EqualFold(strings.TrimSpace(answer), strings.TrimSpace(currentQ.Answer))

	status := "red"
	if isCorrect {
		status = "green"
		state.CorrectCount++
	} else {
		state.WrongCount++
	}

	state.LettersState[currentQ.Letter] = status
	feedback := models.FeedbackPayload{
		Letter:  currentQ.Letter,
		Status:  status,
		Correct: isCorrect,
	}

	gm.moveToNext(state)
	return state, feedback, true
}

func (gm *GameManager) Pass(sessionID string) (*models.GameState, models.FeedbackPayload, bool) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	state, ok := gm.Sessions[sessionID]
	if !ok || state.IsGameOver {
		return nil, models.FeedbackPayload{}, false
	}

	gm.updateTimer(state)
	if state.IsGameOver {
		return state, models.FeedbackPayload{}, true
	}

	currentQ := state.Questions[state.CurrentIndex]
	state.LettersState[currentQ.Letter] = "yellow"
	state.PendingPasses = append(state.PendingPasses, state.CurrentIndex)

	feedback := models.FeedbackPayload{
		Letter: currentQ.Letter,
		Status: "yellow",
	}

	gm.moveToNext(state)
	return state, feedback, true
}

func (gm *GameManager) moveToNext(state *models.GameState) {
	if state.IsGameOver {
		return
	}

	// If we are in the first round (not yet finished all letters)
	if !state.IsRevisiting {
		state.CurrentIndex++
		if state.CurrentIndex >= len(state.Questions) {
			state.IsRevisiting = true
		}
	}

	// If we are revisiting or just finished first round
	if state.IsRevisiting {
		if len(state.PendingPasses) == 0 {
			state.IsGameOver = true
			state.PassedCount = 0 // They are all resolved or remaining
			return
		}
		// Take the first pending pass
		state.CurrentIndex = state.PendingPasses[0]
		state.PendingPasses = state.PendingPasses[1:]
	}
}

func (gm *GameManager) updateTimer(state *models.GameState) {
	if state.IsGameOver {
		return
	}
	elapsed := int(time.Since(state.StartTime).Seconds())
	state.TimeRemaining = GameDuration - elapsed
	if state.TimeRemaining <= 0 {
		state.TimeRemaining = 0
		state.IsGameOver = true
		// Count remaining questions as passed? Or just leave them.
		// Requirement says "immediately end".
	}
}

func (gm *GameManager) GetState(sessionID string) (*models.GameState, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	state, ok := gm.Sessions[sessionID]
	if !ok {
		return nil, false
	}
	gm.updateTimer(state)
	return state, true
}
