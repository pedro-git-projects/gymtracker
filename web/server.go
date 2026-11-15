// Package web serves the HTML frontend. Every page is rendered on the server
// with html/template; the only client side script is a dependency free rest
// timer that the pages work without.
package web

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pedro-git-projects/gymtracker/model"
	"github.com/pedro-git-projects/gymtracker/store"
)

//go:embed templates/* static/*
var files embed.FS

const userCookie = "gt_user"

type Server struct {
	store *store.Store
	mux   *http.ServeMux
	pages map[string]*template.Template
}

func New(st *store.Store) (*Server, error) {
	s := &Server{store: st, mux: http.NewServeMux(), pages: map[string]*template.Template{}}
	if err := s.parseTemplates(); err != nil {
		return nil, err
	}
	s.routes()
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) parseTemplates() error {
	funcs := template.FuncMap{
		"days":     model.AllDays,
		"targets":  model.AllTargetMuscles,
		"plates":   func(load uint16) string { return model.GreedyCombination(load).String() },
		"secs":     func(d time.Duration) int { return int(d.Seconds()) },
		"dur":      formatDuration,
		"date":     func(t time.Time) string { return t.Format("Mon 02 Jan") },
		"datetime": func(t time.Time) string { return t.Format("Mon 02 Jan 15:04") },
		"clock":    func(t time.Time) string { return t.Format("15:04") },
		"add":      func(a, b int) int { return a + b },
		"sub":      func(a, b int) int { return a - b },
		"mul100":   func(a, b int) int { return a * 100 / b },
		"seq": func(n int) []int {
			out := make([]int, n)
			for i := range out {
				out[i] = i
			}
			return out
		},
		"esc":       url.PathEscape,
		"hasPrefix": strings.HasPrefix,
	}
	entries, err := fs.ReadDir(files, "templates")
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if name == "layout.html" {
			continue
		}
		t, err := template.New("layout.html").Funcs(funcs).ParseFS(files, "templates/layout.html", "templates/"+name)
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		s.pages[name] = t
	}
	return nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	case m > 0 && sec > 0:
		return fmt.Sprintf("%dm%02ds", m, sec)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	default:
		return fmt.Sprintf("%ds", sec)
	}
}

// page is what every template receives.
type page struct {
	Title  string
	Path   string
	UserID string
	User   *store.UserData
	Today  model.Day
	Now    time.Time
	Error  string
	Info   string
	Data   any
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, name, title string, uid string, u *store.UserData, data any) {
	t, ok := s.pages[name]
	if !ok {
		http.Error(w, "missing template "+name, http.StatusInternalServerError)
		return
	}
	now := time.Now()
	p := page{
		Title:  title,
		Path:   r.URL.Path,
		UserID: uid,
		User:   u,
		Today:  model.DayOf(now),
		Now:    now,
		Error:  r.URL.Query().Get("err"),
		Info:   r.URL.Query().Get("ok"),
		Data:   data,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout.html", p); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

func redirect(w http.ResponseWriter, r *http.Request, to string, err error, info string) {
	q := url.Values{}
	if err != nil {
		q.Set("err", err.Error())
	}
	if info != "" {
		q.Set("ok", info)
	}
	if len(q) > 0 {
		to += "?" + q.Encode()
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
}

// --- routing -------------------------------------------------------------

func (s *Server) routes() {
	static, _ := fs.Sub(files, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	s.mux.HandleFunc("GET /{$}", s.dashboard)
	s.mux.HandleFunc("POST /users", s.createUser)
	s.mux.HandleFunc("POST /users/select", s.selectUser)
	s.mux.HandleFunc("POST /users/switch", s.switchUser)

	s.mux.HandleFunc("GET /programs", s.programs)
	s.mux.HandleFunc("POST /programs", s.createProgram)
	s.mux.HandleFunc("GET /programs/{id}", s.program)
	s.mux.HandleFunc("POST /programs/{id}/rename", s.renameProgram)
	s.mux.HandleFunc("POST /programs/{id}/activate", s.activateProgram)
	s.mux.HandleFunc("POST /programs/{id}/delete", s.deleteProgram)
	s.mux.HandleFunc("POST /programs/{id}/days/{day}/target", s.setTarget)
	s.mux.HandleFunc("POST /programs/{id}/days/{day}/clear", s.clearDay)
	s.mux.HandleFunc("POST /programs/{id}/days/{day}/exercises", s.addExercise)
	s.mux.HandleFunc("POST /programs/{id}/days/{day}/exercises/{idx}/delete", s.deleteExercise)
	s.mux.HandleFunc("POST /programs/{id}/days/{day}/exercises/{idx}/move", s.moveExercise)

	s.mux.HandleFunc("GET /session", s.session)
	s.mux.HandleFunc("POST /session/start", s.startSession)
	s.mux.HandleFunc("POST /session/sets", s.logSet)
	s.mux.HandleFunc("POST /session/load", s.setLoad)
	s.mux.HandleFunc("POST /session/notes", s.setNotes)
	s.mux.HandleFunc("POST /session/finish", s.finishSession)
	s.mux.HandleFunc("POST /session/discard", s.discardSession)

	s.mux.HandleFunc("GET /history", s.history)
	s.mux.HandleFunc("GET /history/{id}", s.historyEntry)
	s.mux.HandleFunc("POST /history/{id}/delete", s.deleteHistoryEntry)

	s.mux.HandleFunc("GET /exercises", s.exercises)
	s.mux.HandleFunc("GET /exercises/{name}", s.exercise)

	s.mux.HandleFunc("GET /plates", s.plates)
}

// --- users ---------------------------------------------------------------

func (s *Server) currentUser(r *http.Request, d *store.Data) (string, *store.UserData) {
	c, err := r.Cookie(userCookie)
	if err != nil {
		return "", nil
	}
	return c.Value, d.Users[c.Value]
}

func setUserCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     userCookie,
		Value:    id,
		Path:     "/",
		MaxAge:   int((365 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteRune('-')
		}
	}
	return b.String()
}

type usersView struct {
	Users []userRow
}

type userRow struct {
	ID   string
	Name string
}

func (s *Server) renderUsers(w http.ResponseWriter, r *http.Request, d *store.Data) {
	var v usersView
	for id, u := range d.Users {
		v.Users = append(v.Users, userRow{ID: id, Name: u.Name})
	}
	sort.Slice(v.Users, func(i, j int) bool { return v.Users[i].Name < v.Users[j].Name })
	s.render(w, r, "users.html", "Who is training?", "", nil, v)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	id := slug(name)
	if id == "" {
		redirect(w, r, "/", errors.New("name cannot be empty"), "")
		return
	}
	err := s.store.Update(func(d *store.Data) error {
		if _, exists := d.Users[id]; exists {
			return fmt.Errorf("profile %q already exists", name)
		}
		d.Users[id] = &store.UserData{Name: name}
		return nil
	})
	if err != nil {
		redirect(w, r, "/", err, "")
		return
	}
	setUserCookie(w, id)
	redirect(w, r, "/programs", nil, "Welcome, "+name+". Create your first program.")
}

func (s *Server) selectUser(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	var ok bool
	s.store.View(func(d *store.Data) { _, ok = d.Users[id] })
	if !ok {
		redirect(w, r, "/", errors.New("unknown profile"), "")
		return
	}
	setUserCookie(w, id)
	redirect(w, r, "/", nil, "")
}

func (s *Server) switchUser(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: userCookie, Value: "", Path: "/", MaxAge: -1})
	redirect(w, r, "/", nil, "")
}

// withUser runs fn with write access to the current user, redirecting to the
// profile picker when there is none.
func (s *Server) withUser(w http.ResponseWriter, r *http.Request, fn func(u *store.UserData) error) error {
	var missing bool
	err := s.store.Update(func(d *store.Data) error {
		_, u := s.currentUser(r, d)
		if u == nil {
			missing = true
			return nil
		}
		return fn(u)
	})
	if missing {
		redirect(w, r, "/", nil, "")
		return errNoUser
	}
	return err
}

var errNoUser = errors.New("no user selected")

// --- dashboard -----------------------------------------------------------

type dashboardView struct {
	Program *model.Program
	Week    []dayCell
	Today   model.Day
	Workout *model.Workout
	Preview []exercisePreview
	Current *model.Session
	Done    int
	Total   int
	Recent  []model.Session
	NextDay *model.Day
}

type exercisePreview struct {
	model.WorkoutExercise
	Load uint16
}

type dayCell struct {
	Day      model.Day
	Training bool
	Done     bool
	Today    bool
	Target   model.TargetMuscle
}

func weekCells(p *model.Program, h model.History, now time.Time) []dayCell {
	progress := h.WeekProgress(p, now)
	cells := make([]dayCell, 0, 7)
	for _, d := range model.AllDays() {
		c := dayCell{Day: d, Today: d == model.DayOf(now)}
		if wk, ok := p.Workout(d); ok {
			c.Training = true
			c.Target = wk.Target
			c.Done = progress[d]
		}
		cells = append(cells, c)
	}
	return cells
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		if u == nil {
			s.renderUsers(w, r, d)
			return
		}
		now := time.Now()
		v := dashboardView{Today: model.DayOf(now), Current: u.Current, Program: u.Active()}
		if v.Current != nil {
			for _, l := range v.Current.Exercises {
				v.Total += len(l.Sets)
				v.Done += len(l.CompletedSets())
			}
		}
		if p := v.Program; p != nil {
			v.Week = weekCells(p, u.History, now)
			if wk, ok := p.Workout(v.Today); ok {
				v.Workout = &wk
				for _, e := range wk.Exercises {
					load := e.Load
					if last, ok := u.History.LastLoad(p.ID, v.Today, e.Exercise); ok {
						load = last
					}
					v.Preview = append(v.Preview, exercisePreview{WorkoutExercise: e, Load: load})
				}
			} else {
				for i := 1; i <= 7; i++ {
					d := model.Day((int(v.Today) + i) % 7)
					if _, ok := p.Workout(d); ok {
						v.NextDay = &d
						break
					}
				}
			}
		}
		sorted := u.History.Sorted()
		for i := len(sorted) - 1; i >= 0 && len(v.Recent) < 5; i-- {
			v.Recent = append(v.Recent, sorted[i])
		}
		s.render(w, r, "dashboard.html", "Today", uid, u, v)
	})
}

// --- programs ------------------------------------------------------------

func (s *Server) programs(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		if u == nil {
			redirect(w, r, "/", nil, "")
			return
		}
		s.render(w, r, "programs.html", "Programs", uid, u, nil)
	})
}

func (s *Server) createProgram(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		redirect(w, r, "/programs", errors.New("program name cannot be empty"), "")
		return
	}
	id := store.NewID()
	err := s.withUser(w, r, func(u *store.UserData) error {
		p := model.NewProgram(name)
		p.ID = id
		u.Programs = append(u.Programs, p)
		if u.ActiveProgram == "" {
			u.ActiveProgram = id
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, errNoUser) {
			redirect(w, r, "/programs", err, "")
		}
		return
	}
	redirect(w, r, "/programs/"+id, nil, "Program created. Add a workout to each training day.")
}

type programView struct {
	Program *model.Program
	Active  bool
	Days    []programDay
}

type programDay struct {
	Day     model.Day
	Workout *model.Workout
}

func (s *Server) program(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		if u == nil {
			redirect(w, r, "/", nil, "")
			return
		}
		p := u.Program(r.PathValue("id"))
		if p == nil {
			http.NotFound(w, r)
			return
		}
		v := programView{Program: p, Active: u.ActiveProgram == p.ID}
		for _, d := range model.AllDays() {
			pd := programDay{Day: d}
			if wk, ok := p.Workout(d); ok {
				pd.Workout = &wk
			}
			v.Days = append(v.Days, pd)
		}
		s.render(w, r, "program.html", p.Name, uid, u, v)
	})
}

// programAction wraps the common "find program, mutate, redirect back" flow.
func (s *Server) programAction(w http.ResponseWriter, r *http.Request, info string, fn func(u *store.UserData, p *model.Program) error) {
	id := r.PathValue("id")
	back := "/programs/" + id
	err := s.withUser(w, r, func(u *store.UserData) error {
		p := u.Program(id)
		if p == nil {
			return errors.New("program not found")
		}
		return fn(u, p)
	})
	if errors.Is(err, errNoUser) {
		return
	}
	redirect(w, r, back, err, info)
}

func (s *Server) renameProgram(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	s.programAction(w, r, "Renamed.", func(_ *store.UserData, p *model.Program) error {
		if name == "" {
			return errors.New("program name cannot be empty")
		}
		p.Name = name
		return nil
	})
}

func (s *Server) activateProgram(w http.ResponseWriter, r *http.Request) {
	s.programAction(w, r, "Program activated.", func(u *store.UserData, p *model.Program) error {
		u.ActiveProgram = p.ID
		return nil
	})
}

func (s *Server) deleteProgram(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := s.withUser(w, r, func(u *store.UserData) error {
		if u.Current != nil && u.Current.ProgramID == id {
			return errors.New("finish or discard the running session first")
		}
		u.RemoveProgram(id)
		return nil
	})
	if errors.Is(err, errNoUser) {
		return
	}
	redirect(w, r, "/programs", err, "Program deleted.")
}

func parseDay(r *http.Request) (model.Day, error) {
	n, err := strconv.Atoi(r.PathValue("day"))
	if err != nil || !model.Day(n).Valid() {
		return 0, errors.New("invalid day")
	}
	return model.Day(n), nil
}

func formUint(r *http.Request, key string, def int) int {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func parseExercise(r *http.Request) (model.WorkoutExercise, error) {
	e := model.WorkoutExercise{
		Exercise: model.Exercise(strings.TrimSpace(r.FormValue("exercise"))),
		Sets:     model.Sets(formUint(r, "sets", 0)),
		Reps:     model.Reps(formUint(r, "reps", 0)),
		RM: model.RM{
			LowerBound: uint8(formUint(r, "rm_low", 0)),
			UpperBound: uint8(formUint(r, "rm_high", 0)),
		},
		Load:  uint16(formUint(r, "load", 0)),
		Rest:  time.Duration(formUint(r, "rest", 0)) * time.Second,
		Notes: strings.TrimSpace(r.FormValue("notes")),
	}
	if e.RM.LowerBound == 0 && e.RM.UpperBound == 0 {
		e.RM = model.RM{LowerBound: uint8(e.Reps), UpperBound: uint8(e.Reps)}
	}
	return e, e.Validate()
}

func (s *Server) setTarget(w http.ResponseWriter, r *http.Request) {
	s.programAction(w, r, "Target updated.", func(_ *store.UserData, p *model.Program) error {
		day, err := parseDay(r)
		if err != nil {
			return err
		}
		t := model.TargetMuscle(formUint(r, "target", -1))
		if !t.Valid() {
			return errors.New("invalid target")
		}
		wk, ok := p.Workout(day)
		if !ok {
			return errors.New("that day has no workout")
		}
		wk.Target = t
		return p.SetWorkout(day, wk)
	})
}

func (s *Server) clearDay(w http.ResponseWriter, r *http.Request) {
	s.programAction(w, r, "Day cleared.", func(_ *store.UserData, p *model.Program) error {
		day, err := parseDay(r)
		if err != nil {
			return err
		}
		p.RemoveWorkout(day)
		return nil
	})
}

func (s *Server) addExercise(w http.ResponseWriter, r *http.Request) {
	s.programAction(w, r, "Exercise added.", func(_ *store.UserData, p *model.Program) error {
		day, err := parseDay(r)
		if err != nil {
			return err
		}
		e, err := parseExercise(r)
		if err != nil {
			return err
		}
		wk, ok := p.Workout(day)
		if !ok {
			t := model.TargetMuscle(formUint(r, "target", -1))
			if !t.Valid() {
				return errors.New("pick a target for the new workout")
			}
			wk = model.Workout{Target: t}
		}
		wk.AddExercise(e)
		return p.SetWorkout(day, wk)
	})
}

func exerciseIndex(r *http.Request, wk model.Workout) (int, error) {
	idx, err := strconv.Atoi(r.PathValue("idx"))
	if err != nil || idx < 0 || idx >= len(wk.Exercises) {
		return 0, errors.New("exercise not found")
	}
	return idx, nil
}

func (s *Server) deleteExercise(w http.ResponseWriter, r *http.Request) {
	s.programAction(w, r, "Exercise removed.", func(_ *store.UserData, p *model.Program) error {
		day, err := parseDay(r)
		if err != nil {
			return err
		}
		wk, ok := p.Workout(day)
		if !ok {
			return errors.New("that day has no workout")
		}
		idx, err := exerciseIndex(r, wk)
		if err != nil {
			return err
		}
		wk.Exercises = append(wk.Exercises[:idx:idx], wk.Exercises[idx+1:]...)
		if len(wk.Exercises) == 0 {
			p.RemoveWorkout(day)
			return nil
		}
		return p.SetWorkout(day, wk)
	})
}

func (s *Server) moveExercise(w http.ResponseWriter, r *http.Request) {
	s.programAction(w, r, "", func(_ *store.UserData, p *model.Program) error {
		day, err := parseDay(r)
		if err != nil {
			return err
		}
		wk, ok := p.Workout(day)
		if !ok {
			return errors.New("that day has no workout")
		}
		idx, err := exerciseIndex(r, wk)
		if err != nil {
			return err
		}
		to := idx - 1
		if r.FormValue("dir") == "down" {
			to = idx + 1
		}
		if to < 0 || to >= len(wk.Exercises) {
			return nil
		}
		wk.Exercises[idx], wk.Exercises[to] = wk.Exercises[to], wk.Exercises[idx]
		return p.SetWorkout(day, wk)
	})
}

// --- session -------------------------------------------------------------

type sessionView struct {
	Session   *model.Session
	Exercises []sessionExercise
	Done      int
	Total     int
}

type sessionExercise struct {
	Index    int
	Log      model.ExerciseLog
	Previous []model.ExerciseEntry
	HitTop   bool
}

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		if u == nil {
			redirect(w, r, "/", nil, "")
			return
		}
		if u.Current == nil {
			redirect(w, r, "/", errors.New("no session running"), "")
			return
		}
		v := sessionView{Session: u.Current}
		for i, l := range u.Current.Exercises {
			prev := u.History.ForExercise(l.Exercise())
			if len(prev) > 3 {
				prev = prev[len(prev)-3:]
			}
			for a, b := 0, len(prev)-1; a < b; a, b = a+1, b-1 {
				prev[a], prev[b] = prev[b], prev[a]
			}
			v.Exercises = append(v.Exercises, sessionExercise{Index: i, Log: l, Previous: prev, HitTop: l.HitTopOfRange()})
			v.Total += len(l.Sets)
			v.Done += len(l.CompletedSets())
		}
		s.render(w, r, "session.html", "Session", uid, u, v)
	})
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	day := model.Day(formUint(r, "day", int(model.DayOf(now))))
	err := s.withUser(w, r, func(u *store.UserData) error {
		if u.Current != nil {
			return errors.New("a session is already running")
		}
		p := u.Active()
		if p == nil {
			return errors.New("no active program")
		}
		sess, err := model.NewSession(p, day, now, u.History)
		if err != nil {
			return err
		}
		sess.ID = store.NewID()
		u.Current = &sess
		return nil
	})
	if errors.Is(err, errNoUser) {
		return
	}
	if err != nil {
		redirect(w, r, "/", err, "")
		return
	}
	redirect(w, r, "/session", nil, "")
}

// sessionAction mutates the running session and redirects back to it.
func (s *Server) sessionAction(w http.ResponseWriter, r *http.Request, fn func(u *store.UserData, sess *model.Session) error) {
	err := s.withUser(w, r, func(u *store.UserData) error {
		if u.Current == nil {
			return errors.New("no session running")
		}
		return fn(u, u.Current)
	})
	if errors.Is(err, errNoUser) {
		return
	}
	anchor := ""
	if ex := r.FormValue("anchor"); ex != "" {
		anchor = "#" + url.PathEscape(ex)
	}
	if err != nil {
		redirect(w, r, "/session", err, "")
		return
	}
	http.Redirect(w, r, "/session"+anchor, http.StatusSeeOther)
}

func (s *Server) logSet(w http.ResponseWriter, r *http.Request) {
	s.sessionAction(w, r, func(_ *store.UserData, sess *model.Session) error {
		ex := model.Exercise(r.FormValue("exercise"))
		idx := formUint(r, "set", -1)
		reps := formUint(r, "reps", 0)
		load := formUint(r, "load", 0)
		if reps <= 0 || reps > 255 {
			return fmt.Errorf("%s: reps must be between 1 and 255", ex)
		}
		return sess.LogSet(ex, idx, model.Reps(reps), uint16(load))
	})
}

func (s *Server) setLoad(w http.ResponseWriter, r *http.Request) {
	s.sessionAction(w, r, func(_ *store.UserData, sess *model.Session) error {
		ex := model.Exercise(r.FormValue("exercise"))
		return sess.SetLoad(ex, uint16(formUint(r, "load", 0)))
	})
}

func (s *Server) setNotes(w http.ResponseWriter, r *http.Request) {
	s.sessionAction(w, r, func(_ *store.UserData, sess *model.Session) error {
		notes := strings.TrimSpace(r.FormValue("notes"))
		if ex := r.FormValue("exercise"); ex != "" {
			l := sess.Exercise(model.Exercise(ex))
			if l == nil {
				return fmt.Errorf("%s is not part of this session", ex)
			}
			l.Notes = notes
			return nil
		}
		sess.Notes = notes
		return nil
	})
}

func (s *Server) finishSession(w http.ResponseWriter, r *http.Request) {
	var id string
	err := s.withUser(w, r, func(u *store.UserData) error {
		if u.Current == nil {
			return errors.New("no session running")
		}
		if notes := strings.TrimSpace(r.FormValue("notes")); notes != "" {
			u.Current.Notes = notes
		}
		u.Current.Finish(time.Now())
		id = u.Current.ID
		u.History = append(u.History, *u.Current)
		u.Current = nil
		return nil
	})
	if errors.Is(err, errNoUser) {
		return
	}
	if err != nil {
		redirect(w, r, "/session", err, "")
		return
	}
	redirect(w, r, "/history/"+id, nil, "Session saved. Good work.")
}

func (s *Server) discardSession(w http.ResponseWriter, r *http.Request) {
	err := s.withUser(w, r, func(u *store.UserData) error {
		u.Current = nil
		return nil
	})
	if errors.Is(err, errNoUser) {
		return
	}
	redirect(w, r, "/", err, "Session discarded.")
}

// --- history -------------------------------------------------------------

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		if u == nil {
			redirect(w, r, "/", nil, "")
			return
		}
		sorted := u.History.Sorted()
		for a, b := 0, len(sorted)-1; a < b; a, b = a+1, b-1 {
			sorted[a], sorted[b] = sorted[b], sorted[a]
		}
		s.render(w, r, "history.html", "History", uid, u, sorted)
	})
}

type historyEntryView struct {
	Session *model.Session
	Program string
}

func (s *Server) historyEntry(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		if u == nil {
			redirect(w, r, "/", nil, "")
			return
		}
		sess := u.Session(r.PathValue("id"))
		if sess == nil {
			http.NotFound(w, r)
			return
		}
		v := historyEntryView{Session: sess}
		if p := u.Program(sess.ProgramID); p != nil {
			v.Program = p.Name
		}
		s.render(w, r, "history_entry.html", sess.Target.String(), uid, u, v)
	})
}

func (s *Server) deleteHistoryEntry(w http.ResponseWriter, r *http.Request) {
	err := s.withUser(w, r, func(u *store.UserData) error {
		u.RemoveSession(r.PathValue("id"))
		return nil
	})
	if errors.Is(err, errNoUser) {
		return
	}
	redirect(w, r, "/history", err, "Session deleted.")
}

// --- exercises -----------------------------------------------------------

type exerciseRow struct {
	Name     model.Exercise
	Sessions int
	TopLoad  uint16
	Last     time.Time
}

func (s *Server) exercises(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		if u == nil {
			redirect(w, r, "/", nil, "")
			return
		}
		rows := map[model.Exercise]*exerciseRow{}
		for _, sess := range u.History {
			for _, l := range sess.Exercises {
				if len(l.CompletedSets()) == 0 {
					continue
				}
				row := rows[l.Exercise()]
				if row == nil {
					row = &exerciseRow{Name: l.Exercise()}
					rows[l.Exercise()] = row
				}
				row.Sessions++
				if l.TopLoad() > row.TopLoad {
					row.TopLoad = l.TopLoad()
				}
				if sess.StartedAt.After(row.Last) {
					row.Last = sess.StartedAt
				}
			}
		}
		for _, p := range u.Programs {
			for _, wk := range p.Workouts {
				for _, e := range wk.Exercises {
					if _, ok := rows[e.Exercise]; !ok {
						rows[e.Exercise] = &exerciseRow{Name: e.Exercise}
					}
				}
			}
		}
		list := make([]exerciseRow, 0, len(rows))
		for _, row := range rows {
			list = append(list, *row)
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		s.render(w, r, "exercises.html", "Exercises", uid, u, list)
	})
}

type exerciseView struct {
	Name    model.Exercise
	Entries []model.ExerciseEntry
	Best    uint16
	MaxVol  int
}

func (s *Server) exercise(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		if u == nil {
			redirect(w, r, "/", nil, "")
			return
		}
		name := model.Exercise(r.PathValue("name"))
		v := exerciseView{Name: name, Entries: u.History.ForExercise(name)}
		for _, e := range v.Entries {
			if e.Log.TopLoad() > v.Best {
				v.Best = e.Log.TopLoad()
			}
			if e.Log.Volume() > v.MaxVol {
				v.MaxVol = e.Log.Volume()
			}
		}
		for a, b := 0, len(v.Entries)-1; a < b; a, b = a+1, b-1 {
			v.Entries[a], v.Entries[b] = v.Entries[b], v.Entries[a]
		}
		s.render(w, r, "exercise.html", string(name), uid, u, v)
	})
}

// --- plates --------------------------------------------------------------

type platesView struct {
	Load         int
	Combinations []model.WeightCombination
	Total        int
	Truncated    bool
}

func (s *Server) plates(w http.ResponseWriter, r *http.Request) {
	s.store.View(func(d *store.Data) {
		uid, u := s.currentUser(r, d)
		v := platesView{Load: formUint(r, "load", 0)}
		if v.Load > 300 {
			v.Load = 300
		}
		if v.Load > 0 {
			all := model.CalculateCombinations(uint16(v.Load))
			v.Total = len(all)
			if len(all) > 40 {
				all = all[:40]
				v.Truncated = true
			}
			v.Combinations = all
		}
		s.render(w, r, "plates.html", "Plates", uid, u, v)
	})
}
