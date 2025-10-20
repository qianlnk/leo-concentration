package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed web/templates/*.html
var templatesFS embed.FS

//go:embed web/static/*
var staticFS embed.FS

var (
	db        *sql.DB
	tpl       *template.Template
	staticSub fs.FS
)

func main() {
	// Ensure data directory
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	var err error
	db, err = sql.Open("sqlite", filepath.Join("data", "leo_concentration.db"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	if err = migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err = seedTrainings(db); err != nil {
		log.Printf("seed trainings: %v", err)
	}

	// Parse templates from embed FS
	var errParse error
	tpl, errParse = template.ParseFS(templatesFS, "web/templates/*.html")
	if errParse != nil {
		log.Fatalf("parse templates: %v", errParse)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/trainings", handleTrainings)
	mux.HandleFunc("/training", handleTraining)
	mux.HandleFunc("/session/start", handleSessionStart)
	mux.HandleFunc("/session", handleSession)
	mux.HandleFunc("/session/update", handleSessionUpdate)
	mux.HandleFunc("/session/finish", handleSessionFinish)
	mux.HandleFunc("/reports/daily", handleReportsDaily)
	mux.HandleFunc("/api/session/update", handleSessionUpdateJSON)
	mux.HandleFunc("/play/balance-hero", handlePlayBalance)
	mux.HandleFunc("/play/number-trace", handlePlayNumberTrace)
	mux.HandleFunc("/play/memory-cards", handlePlayMemory)
	mux.HandleFunc("/play/poem-trace", handlePlayPoemTrace)
	mux.HandleFunc("/play/warmup-jumping", handlePlayWarmup)
	mux.HandleFunc("/play/color-match", handlePlayColorMatch)
	mux.HandleFunc("/play/eagle-eye", handlePlayEagleEye)
	mux.HandleFunc("/play/cup-ball", handlePlayCupBall)

	// Static files (serve from embedded web/static)
	var subErr error
	staticSub, subErr = fs.Sub(staticFS, "web/static")
	if subErr != nil {
		log.Printf("static fs sub: %v", subErr)
		staticSub = staticFS
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	addr := ":8080"
	log.Printf("Server running at %s", addr)
	log.Fatal(http.ListenAndServe(addr, logRequest(mux)))
}

func logRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %dms", r.Method, r.URL.Path, time.Since(start).Milliseconds())
	})
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS trainings (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            slug TEXT,
            description TEXT,
            category TEXT,
            suggested_minutes INTEGER DEFAULT 10,
            active INTEGER DEFAULT 1
        );`,
		`CREATE TABLE IF NOT EXISTS sessions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            training_id INTEGER NOT NULL,
            started_at DATETIME NOT NULL,
            ended_at DATETIME,
            duration_seconds INTEGER,
            success_count INTEGER DEFAULT 0,
            error_count INTEGER DEFAULT 0,
            notes TEXT,
            FOREIGN KEY(training_id) REFERENCES trainings(id)
        );`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate exec: %w", err)
		}
	}
	// Ensure slug column exists for existing DBs
	var hasSlug bool
	rows, err := db.Query(`PRAGMA table_info(trainings)`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt sql.NullString
			_ = rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
			if strings.EqualFold(name, "slug") {
				hasSlug = true
				break
			}
		}
	}
	if !hasSlug {
		if _, err := db.Exec(`ALTER TABLE trainings ADD COLUMN slug TEXT`); err != nil {
			log.Printf("alter add slug: %v", err)
		}
	}
	return nil
}

func seedTrainings(db *sql.DB) error {
	// Ensure core trainings exist; insert if missing by name.
	// Backfill slug for existing rows without slug.
	ensure := func(name, slug, desc, cat string, min int) error {
		var exists int
		if err := db.QueryRow(`SELECT COUNT(1) FROM trainings WHERE name=?`, name).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			_, err := db.Exec(`INSERT INTO trainings(name, slug, description, category, suggested_minutes, active) VALUES(?,?,?,?,?,1)`, name, slug, desc, cat, min)
			return err
		}
		// backfill slug if missing
		if _, err := db.Exec(`UPDATE trainings SET slug=? WHERE name=? AND (slug IS NULL OR slug='')`, slug, name); err != nil {
			return err
		}
		return nil
	}
	if err := ensure("平衡超人", "balance-hero", "左右脚单独站立，提升身体控制与专注。", "身体协调", 10); err != nil {
		return err
	}
	if err := ensure("数字迷踪", "number-trace", "按顺序点击数字网格，训练注意力与顺序记忆。", "数字/认知", 10); err != nil {
		return err
	}
	if err := ensure("记忆卡片", "memory-cards", "翻牌记忆训练，找到两张相同的卡片。", "记忆", 8); err != nil {
		return err
	}
	if err := ensure("唐诗迷踪", "poem-trace", "按顺序点击诗句的字，训练顺序与记忆。", "语文/认知", 10); err != nil {
		return err
	}
	if err := ensure("全身唤醒", "warmup-jumping", "做20个开合跳，让全身都充满能量！", "身体激活", 5); err != nil {
		return err
	}
	if err := ensure("颜色对对碰", "color-match", "识别颜色字的字义，而非字体颜色，训练抗干扰能力。", "认知/专注", 8); err != nil {
		return err
	}
	if err := ensure("火眼金睛", "eagle-eye", "在众多物品中快速找出指定目标，训练观察力与专注力。", "观察/专注", 10); err != nil {
		return err
	}
	if err := ensure("眼疾手快", "cup-ball", "观察球在哪个杯子下，记住杯子交换过程，考验记忆力与专注力。", "观察/记忆", 8); err != nil {
		return err
	}
	return nil
}

// Helpers
func parseInt(v string) (int, error) { return strconv.Atoi(strings.TrimSpace(v)) }

// Handlers
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	type Training struct {
		ID             int
		Name, Category string
		Suggested      int
	}
	rows, err := db.Query("SELECT id, name, category, suggested_minutes FROM trainings WHERE active=1 ORDER BY id")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var list []Training
	for rows.Next() {
		var t Training
		if err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.Suggested); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		list = append(list, t)
	}
	// Today summary
	today := time.Now().Format("2006-01-02")
	var totalSec int
	_ = db.QueryRow(`SELECT COALESCE(SUM(duration_seconds),0) FROM sessions WHERE date(started_at)=?`, today).Scan(&totalSec)
	renderTemplate(w, "index.html", map[string]any{"Trainings": list, "TotalSecs": totalSec, "TotalMin": totalSec / 60, "Today": today})
}

func handleTrainings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		type T struct {
			ID                          int
			Name, Category, Description string
			Suggested                   int
			Active                      int
		}
		rows, err := db.Query("SELECT id, name, category, description, suggested_minutes, active FROM trainings ORDER BY id")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()
		var list []T
		for rows.Next() {
			var t T
			if err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.Description, &t.Suggested, &t.Active); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			list = append(list, t)
		}
		renderTemplate(w, "trainings.html", map[string]any{"Trainings": list})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		category := strings.TrimSpace(r.FormValue("category"))
		description := strings.TrimSpace(r.FormValue("description"))
		suggested, _ := parseInt(r.FormValue("suggested"))
		if name == "" {
			http.Error(w, "name required", 400)
			return
		}
		if suggested <= 0 {
			suggested = 10
		}
		if _, err := db.Exec(`INSERT INTO trainings(name, category, description, suggested_minutes, active) VALUES(?,?,?,?,1)`, name, category, description, suggested); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/trainings", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTraining(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var name, category, description string
	var suggested int
	err := db.QueryRow("SELECT name, category, description, suggested_minutes FROM trainings WHERE id=?", id).Scan(&name, &category, &description, &suggested)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	renderTemplate(w, "training.html", map[string]any{"ID": id, "Name": name, "Category": category, "Description": description, "Suggested": suggested})
}

func handleSessionStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	tid, _ := parseInt(r.FormValue("training_id"))
	if tid == 0 {
		http.Error(w, "missing training_id", 400)
		return
	}
	now := time.Now()
	res, err := db.Exec(`INSERT INTO sessions(training_id, started_at) VALUES(?,?)`, tid, now)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	id64, _ := res.LastInsertId()
	var slug sql.NullString
	_ = db.QueryRow(`SELECT slug FROM trainings WHERE id=?`, tid).Scan(&slug)
	if slug.Valid && slug.String != "" {
		http.Redirect(w, r, fmt.Sprintf("/play/%s?id=%d", slug.String, id64), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/session?id=%d", id64), http.StatusSeeOther)
}

func handleSession(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	var startedAt, endedAt sql.NullTime
	var success, errorCnt int
	err := db.QueryRow(`SELECT training_id, started_at, ended_at, success_count, error_count FROM sessions WHERE id=?`, id).Scan(&trainingID, &startedAt, &endedAt, &success, &errorCnt)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	var tName string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&tName)
	renderTemplate(w, "session.html", map[string]any{"ID": id, "TrainingName": tName, "Started": startedAt.Time.Format("15:04:05"), "Ended": func() string {
		if endedAt.Valid {
			return endedAt.Time.Format("15:04:05")
		}
		return ""
	}(), "Success": success, "Errors": errorCnt})
}

func handleSessionUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	id, _ := parseInt(r.FormValue("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	success, _ := parseInt(r.FormValue("success"))
	errors, _ := parseInt(r.FormValue("errors"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	if _, err := db.Exec(`UPDATE sessions SET success_count=?, error_count=?, notes=? WHERE id=?`, success, errors, notes, id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/session?id=%d", id), http.StatusSeeOther)
}

// JSON/form lightweight API for in-game updates
func handleSessionUpdateJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	id, _ := parseInt(r.FormValue("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	success, _ := parseInt(r.FormValue("success"))
	errors, _ := parseInt(r.FormValue("errors"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	appendFlag := strings.TrimSpace(r.FormValue("append"))
	if appendFlag == "1" && notes != "" {
		var old sql.NullString
		_ = db.QueryRow(`SELECT notes FROM sessions WHERE id=?`, id).Scan(&old)
		if old.Valid && old.String != "" {
			notes = old.String + "\n" + notes
		}
	}
	if _, err := db.Exec(`UPDATE sessions SET success_count=?, error_count=?, notes=? WHERE id=?`, success, errors, notes, id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// Play pages
func handlePlayBalance(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_balance.html", map[string]any{"ID": id, "TrainingName": name})
}

func handlePlayNumberTrace(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_number_trace.html", map[string]any{"ID": id, "TrainingName": name})
}

func handlePlayMemory(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_memory_cards.html", map[string]any{"ID": id, "TrainingName": name})
}

func handlePlayPoemTrace(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_poem_trace.html", map[string]any{"ID": id, "TrainingName": name})
}

func handlePlayWarmup(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_warmup.html", map[string]any{"ID": id, "TrainingName": name})
}

// handlePlayColorMatch 处理颜色对对碰游戏页面
func handlePlayColorMatch(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_color_match.html", map[string]any{"ID": id, "TrainingName": name})
}

// handlePlayEagleEye 处理火眼金睛游戏页面
func handlePlayEagleEye(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_eagle_eye.html", map[string]any{"ID": id, "TrainingName": name})
}

// handlePlayCupBall 处理眼疾手快游戏页面
func handlePlayCupBall(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_cup_ball.html", map[string]any{"ID": id, "TrainingName": name})
}

func handleSessionFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	id, _ := parseInt(r.FormValue("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	ended := time.Now()
	var started time.Time
	if err := db.QueryRow(`SELECT started_at FROM sessions WHERE id=?`, id).Scan(&started); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	dur := int(ended.Sub(started).Seconds())
	if _, err := db.Exec(`UPDATE sessions SET ended_at=?, duration_seconds=? WHERE id=?`, ended, dur, id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleReportsDaily(w http.ResponseWriter, r *http.Request) {
	type Day struct {
		Day      string
		TotalSec int
		TotalMin int
		Sessions int
	}
	rows, err := db.Query(`SELECT date(started_at) d, COALESCE(SUM(duration_seconds),0) s, COUNT(1) c FROM sessions WHERE ended_at IS NOT NULL GROUP BY d ORDER BY d DESC LIMIT 30`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var list []Day
	for rows.Next() {
		var d Day
		if err := rows.Scan(&d.Day, &d.TotalSec, &d.Sessions); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		d.TotalMin = d.TotalSec / 60
		list = append(list, d)
	}
	renderTemplate(w, "reports_daily.html", map[string]any{"Days": list})
}

// Template rendering
func renderTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
