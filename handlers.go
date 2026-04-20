package main

import (
	"crypto/subtle"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	db   *pgxpool.Pool
	cfg  Config
	tmpl *template.Template
}

// POST /questions
func (s *Server) handleCreateQuestion(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token != s.cfg.AuthToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Question == "" || len(req.Options) == 0 {
		http.Error(w, "question and options are required", http.StatusBadRequest)
		return
	}

	var id uuid.UUID
	err := s.db.QueryRow(r.Context(),
		`INSERT INTO questions (question, options) VALUES ($1, $2) RETURNING id`,
		req.Question, req.Options,
	).Scan(&id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"interaction_url": s.cfg.BaseURL + "/i/" + id.String(),
		"poll_url":        s.cfg.BaseURL + "/poll/" + id.String(),
	})
}

// GET /i/{uuid}
func (s *Server) handleInteractionPage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	var question string
	var options []string
	var answer *string
	err := s.db.QueryRow(r.Context(),
		`SELECT question, options, answer FROM questions WHERE id = $1`, id,
	).Scan(&question, &options, &answer)
	if err == pgx.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := TemplateData{
		UUID:     id.String(),
		Question: question,
		Options:  options,
	}
	if answer != nil {
		data.Answer = *answer
		data.Answered = true
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.tmpl.ExecuteTemplate(w, "interaction", data)
}

// POST /i/{uuid}/respond
func (s *Server) handleRespond(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	answer := r.FormValue("answer")
	if answer == "" {
		http.Error(w, "answer is required", http.StatusBadRequest)
		return
	}

	tag, err := s.db.Exec(r.Context(),
		`UPDATE questions SET answer = $1, answered_at = NOW() WHERE id = $2 AND answer IS NULL`,
		answer, id,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if tag.RowsAffected() == 0 {
		// Distinguish 404 from 409.
		var exists bool
		s.db.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM questions WHERE id = $1)`, id,
		).Scan(&exists)
		if !exists {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "already answered", http.StatusConflict)
		return
	}

	http.Redirect(w, r, "/i/"+id.String(), http.StatusSeeOther)
}

// GET /poll/{uuid}
func (s *Server) handlePoll(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	var answer *string
	err := s.db.QueryRow(r.Context(),
		`SELECT answer FROM questions WHERE id = $1`, id,
	).Scan(&answer)
	if err == pgx.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"answer": answer})
}

const adminCookie = "admin_token"

// GET /admin/login
func (s *Server) handleAdminLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.tmpl.ExecuteTemplate(w, "login", map[string]bool{"Invalid": false})
}

// POST /admin/login
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	token := r.FormValue("token")
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.AuthToken)) != 1 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		s.tmpl.ExecuteTemplate(w, "login", map[string]bool{"Invalid": true})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminCookie,
		Value:    token,
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// GET /admin
func (s *Server) handleAdminSummary(w http.ResponseWriter, r *http.Request) {
	if !s.isAdminAuthenticated(r) {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}

	rows, err := s.db.Query(r.Context(),
		`SELECT id, question, options, answer, answered_at, created_at
		 FROM questions
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var questions []QuestionRow
	for rows.Next() {
		var q QuestionRow
		var id uuid.UUID
		var answeredAt *time.Time
		if err := rows.Scan(&id, &q.Question, &q.Options, &q.Answer, &answeredAt, &q.CreatedAt); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		q.ID = id.String()
		q.AnsweredAt = answeredAt
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.tmpl.ExecuteTemplate(w, "admin", AdminData{Questions: questions})
}

func (s *Server) isAdminAuthenticated(r *http.Request) bool {
	c, err := r.Cookie(adminCookie)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(s.cfg.AuthToken)) == 1
}

func parseUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := chi.URLParam(r, "uuid")
	id, err := uuid.Parse(raw)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return uuid.UUID{}, false
	}
	return id, true
}
