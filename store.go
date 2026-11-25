package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("tarefa não encontrada")
	ErrInvalid  = errors.New("dados inválidos")
)

// Prioridades: 1 = urgente, 2 = alta, 3 = média, 4 = baixa.
const (
	minPriority     = 1
	maxPriority     = 4
	defaultPriority = 3
	maxTitleLen     = 300
)

type Task struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Priority  int        `json:"priority"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"createdAt"`
	DoneAt    *time.Time `json:"doneAt,omitempty"`
}

// Patch traz só os campos que devem mudar.
type Patch struct {
	Title    *string `json:"title"`
	Priority *int    `json:"priority"`
	Done     *bool   `json:"done"`
}

type fileData struct {
	NextID int64  `json:"nextId"`
	Tasks  []Task `json:"tasks"`
}

// Store guarda as tarefas em memória e grava tudo em um arquivo JSON a cada mudança.
type Store struct {
	mu   sync.Mutex
	path string
	data fileData
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, data: fileData{NextID: 1, Tasks: []Task{}}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, fmt.Errorf("arquivo de dados corrompido: %w", err)
	}
	if s.data.Tasks == nil {
		s.data.Tasks = []Task{}
	}
	return s, nil
}

func (s *Store) List() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Task{}, s.data.Tasks...)
}

func (s *Store) Create(title string, priority int) (Task, error) {
	title, err := cleanTitle(title)
	if err != nil {
		return Task{}, err
	}
	if priority == 0 {
		priority = defaultPriority
	}
	if err := checkPriority(priority); err != nil {
		return Task{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	t := Task{ID: s.data.NextID, Title: title, Priority: priority, CreatedAt: time.Now().UTC()}
	prev := s.data
	s.data.NextID++
	s.data.Tasks = append(append([]Task{}, s.data.Tasks...), t)
	if err := s.save(); err != nil {
		s.data = prev
		return Task{}, err
	}
	return t, nil
}

func (s *Store) Update(id int64, p Patch) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(id)
	if i < 0 {
		return Task{}, ErrNotFound
	}
	t := s.data.Tasks[i]
	if p.Title != nil {
		title, err := cleanTitle(*p.Title)
		if err != nil {
			return Task{}, err
		}
		t.Title = title
	}
	if p.Priority != nil {
		if err := checkPriority(*p.Priority); err != nil {
			return Task{}, err
		}
		t.Priority = *p.Priority
	}
	if p.Done != nil && *p.Done != t.Done {
		t.Done = *p.Done
		t.DoneAt = nil
		if t.Done {
			now := time.Now().UTC()
			t.DoneAt = &now
		}
	}
	old := s.data.Tasks[i]
	s.data.Tasks[i] = t
	if err := s.save(); err != nil {
		s.data.Tasks[i] = old
		return Task{}, err
	}
	return t, nil
}

func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	prev := s.data.Tasks
	next := append([]Task{}, prev[:i]...)
	s.data.Tasks = append(next, prev[i+1:]...)
	if err := s.save(); err != nil {
		s.data.Tasks = prev
		return err
	}
	return nil
}

func (s *Store) index(id int64) int {
	for i, t := range s.data.Tasks {
		if t.ID == id {
			return i
		}
	}
	return -1
}

// save grava em arquivo temporário e renomeia, para nunca deixar o JSON pela metade.
func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func cleanTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("%w: o título é obrigatório", ErrInvalid)
	}
	if len([]rune(title)) > maxTitleLen {
		return "", fmt.Errorf("%w: título com mais de %d caracteres", ErrInvalid, maxTitleLen)
	}
	return title, nil
}

func checkPriority(p int) error {
	if p < minPriority || p > maxPriority {
		return fmt.Errorf("%w: prioridade deve ser de %d a %d", ErrInvalid, minPriority, maxPriority)
	}
	return nil
}
