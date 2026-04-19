package history

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Entry represents a single file state snapshot
type Entry struct {
	File      string    `json:"file"`
	Content   []byte    `json:"-"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"` // "patch", "add-block", "remove-block"
	Detail    string    `json:"detail"` // human-readable description
}

// Info is the serializable summary of an entry (without file content)
type Info struct {
	File      string    `json:"file"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
}

// Status reports the current undo/redo state
type Status struct {
	UndoCount int    `json:"undoCount"`
	RedoCount int    `json:"redoCount"`
	UndoStack []Info `json:"undoStack"`
	RedoStack []Info `json:"redoStack"`
}

// Store tracks file version history for a workspace, enabling undo/redo
type Store struct {
	mu        sync.Mutex
	undoStack []Entry
	redoStack []Entry
	maxSize   int
}

// NewStore creates a history store
func NewStore(maxSize int) *Store {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &Store{maxSize: maxSize}
}

// SaveBefore captures the current file content before a mutation.
// Call this BEFORE writing the file.
func (s *Store) SaveBefore(workspacePath, file, action, detail string) error {
	fullPath := fmt.Sprintf("%s/%s", workspacePath, file)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		// File might not exist yet (add-block to new file) — that's OK
		content = nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.undoStack = append(s.undoStack, Entry{
		File:      file,
		Content:   content,
		Timestamp: time.Now(),
		Action:    action,
		Detail:    detail,
	})

	// Trim oldest if over limit
	if len(s.undoStack) > s.maxSize {
		s.undoStack = s.undoStack[len(s.undoStack)-s.maxSize:]
	}

	// Clear redo stack on new change (standard undo/redo behavior)
	s.redoStack = nil

	return nil
}

// Undo reverts the most recent change. Returns the file that was restored.
func (s *Store) Undo(workspacePath string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.undoStack) == 0 {
		return "", fmt.Errorf("nothing to undo")
	}

	// Pop from undo stack
	entry := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]

	fullPath := fmt.Sprintf("%s/%s", workspacePath, entry.File)

	// Save current state to redo stack before restoring
	currentContent, err := os.ReadFile(fullPath)
	if err != nil {
		currentContent = nil
	}
	s.redoStack = append(s.redoStack, Entry{
		File:      entry.File,
		Content:   currentContent,
		Timestamp: time.Now(),
		Action:    entry.Action,
		Detail:    "redo: " + entry.Detail,
	})

	// Restore the previous content
	if entry.Content == nil {
		// File didn't exist before — remove it
		os.Remove(fullPath)
	} else {
		if err := os.WriteFile(fullPath, entry.Content, 0644); err != nil {
			return "", fmt.Errorf("restoring file: %w", err)
		}
	}

	return entry.File, nil
}

// Redo re-applies the most recently undone change.
func (s *Store) Redo(workspacePath string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.redoStack) == 0 {
		return "", fmt.Errorf("nothing to redo")
	}

	// Pop from redo stack
	entry := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]

	fullPath := fmt.Sprintf("%s/%s", workspacePath, entry.File)

	// Save current state to undo stack before re-applying
	currentContent, err := os.ReadFile(fullPath)
	if err != nil {
		currentContent = nil
	}
	s.undoStack = append(s.undoStack, Entry{
		File:      entry.File,
		Content:   currentContent,
		Timestamp: time.Now(),
		Action:    entry.Action,
		Detail:    entry.Detail,
	})

	// Apply the redo content
	if entry.Content == nil {
		os.Remove(fullPath)
	} else {
		if err := os.WriteFile(fullPath, entry.Content, 0644); err != nil {
			return "", fmt.Errorf("restoring file: %w", err)
		}
	}

	return entry.File, nil
}

// GetStatus returns the current undo/redo counts and descriptions
func (s *Store) GetStatus() Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := Status{
		UndoCount: len(s.undoStack),
		RedoCount: len(s.redoStack),
	}

	for _, e := range s.undoStack {
		status.UndoStack = append(status.UndoStack, Info{
			File:      e.File,
			Timestamp: e.Timestamp,
			Action:    e.Action,
			Detail:    e.Detail,
		})
	}
	for _, e := range s.redoStack {
		status.RedoStack = append(status.RedoStack, Info{
			File:      e.File,
			Timestamp: e.Timestamp,
			Action:    e.Action,
			Detail:    e.Detail,
		})
	}

	return status
}
