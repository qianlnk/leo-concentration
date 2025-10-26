package main

import (
	"database/sql"
	"embed"
	"encoding/json"
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
	mux.HandleFunc("/session/finish", handleSessionFinish)
	mux.HandleFunc("/reports/daily", handleReportsDaily)
	mux.HandleFunc("/reports/daily/v2", handleReportsDailyV2)
	mux.HandleFunc("/training/stats/", handleTrainingStats)
	mux.HandleFunc("/test/cup-ball", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "test_cup_ball.html", nil)
	})
	mux.HandleFunc("/api/session/round/start", handleRoundStart)
	mux.HandleFunc("/api/session/round/finish", handleRoundFinish)
	mux.HandleFunc("/api/stats/daily", handleStatsDaily)
	mux.HandleFunc("/api/stats/training/", handleStatsTraining)
	mux.HandleFunc("/play/balance-hero", handlePlayBalance)
	mux.HandleFunc("/play/number-trace", handlePlayNumberTrace)
	mux.HandleFunc("/play/memory-cards", handlePlayMemory)
	mux.HandleFunc("/play/poem-trace", handlePlayPoemTrace)
	mux.HandleFunc("/play/warmup-jumping", handlePlayWarmup)
	mux.HandleFunc("/play/color-match", handlePlayColorMatch)
	mux.HandleFunc("/play/eagle-eye", handlePlayEagleEye)
	mux.HandleFunc("/play/cup-ball", handlePlayCupBall)
	mux.HandleFunc("/play/fish-adventure", handlePlayFishAdventure)

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
            level TEXT,
            status TEXT DEFAULT 'in_progress',
            completed_rounds INTEGER DEFAULT 0,
            total_rounds INTEGER DEFAULT 0,
            FOREIGN KEY(training_id) REFERENCES trainings(id)
        );`,
		`CREATE TABLE IF NOT EXISTS session_rounds (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            session_id INTEGER NOT NULL,
            round_number INTEGER NOT NULL,
            started_at DATETIME NOT NULL,
            ended_at DATETIME,
            duration_seconds INTEGER,
            success BOOLEAN DEFAULT 0,
            score INTEGER DEFAULT 0,
            accuracy REAL DEFAULT 0,
            metadata TEXT,
            FOREIGN KEY(session_id) REFERENCES sessions(id)
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

	// 迁移sessions表：添加新字段
	addColumnIfNotExists := func(table, column, definition string) {
		var hasCol bool
		r, e := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
		if e == nil {
			defer r.Close()
			for r.Next() {
				var cid int
				var name, ctype string
				var notnull, pk int
				var dflt sql.NullString
				_ = r.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
				if strings.EqualFold(name, column) {
					hasCol = true
					break
				}
			}
		}
		if !hasCol {
			sql := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, definition)
			if _, err := db.Exec(sql); err != nil {
				log.Printf("alter add %s.%s: %v", table, column, err)
			}
		}
	}

	addColumnIfNotExists("sessions", "level", "TEXT")
	addColumnIfNotExists("sessions", "status", "TEXT DEFAULT 'in_progress'")
	addColumnIfNotExists("sessions", "completed_rounds", "INTEGER DEFAULT 0")
	addColumnIfNotExists("sessions", "total_rounds", "INTEGER DEFAULT 0")

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
	if err := ensure("小鱼历险记", "fish-adventure", "根据规则吞噬正确的鱼，训练规则记忆、抑制控制与认知灵活性。", "专注/认知", 10); err != nil {
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
		TodayMinutes   int // 今天训练的分钟数
		TodayRounds    int // 今天训练的轮次数
	}

	// 获取今天的日期
	today := time.Now().Format("2006-01-02")

	// 使用LEFT JOIN查询所有训练项目及其今天的统计数据
	query := `
		SELECT
			t.id,
			t.name,
			t.category,
			t.suggested_minutes,
			COALESCE(SUM(sr.duration_seconds), 0) as today_seconds,
			COALESCE(COUNT(sr.id), 0) as today_rounds
		FROM trainings t
		LEFT JOIN sessions s ON t.id = s.training_id AND substr(s.started_at, 1, 10) = ?
		LEFT JOIN session_rounds sr ON s.id = sr.session_id
		WHERE t.active = 1
		GROUP BY t.id, t.name, t.category, t.suggested_minutes
		ORDER BY t.id
	`

	rows, err := db.Query(query, today)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var list []Training
	for rows.Next() {
		var t Training
		var todaySeconds int
		if err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.Suggested, &todaySeconds, &t.TodayRounds); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		t.TodayMinutes = todaySeconds / 60
		list = append(list, t)
	}

	// 计算今日总训练时长
	var totalSec int
	for _, t := range list {
		totalSec += t.TodayMinutes * 60
	}

	renderTemplate(w, "index.html", map[string]any{
		"Trainings": list,
		"TotalSecs": totalSec,
		"TotalMin":  totalSec / 60,
		"Today":     today,
	})
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

func handlePlayFishAdventure(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_fish_adventure.html", map[string]any{"ID": id, "TrainingName": name})
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

// handleReportsDailyV2 显示增强版每日报表页面
func handleReportsDailyV2(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "reports_daily_v2.html", nil)
}

// handleTrainingStats 显示训练项目统计页面
func handleTrainingStats(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取training_id
	path := r.URL.Path
	idStr := strings.TrimPrefix(path, "/training/stats/")
	trainingID, err := parseInt(idStr)
	if err != nil || trainingID == 0 {
		http.Error(w, "invalid training id", 400)
		return
	}

	// 获取训练项目信息
	var trainingName string
	err = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&trainingName)
	if err != nil {
		http.Error(w, "training not found", 404)
		return
	}

	renderTemplate(w, "training_stats.html", map[string]any{
		"TrainingID":   trainingID,
		"TrainingName": trainingName,
	})
}

// handleRoundStart 开始新一轮训练
func handleRoundStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	sessionID, _ := parseInt(r.FormValue("session_id"))
	roundNumber, _ := parseInt(r.FormValue("round_number"))
	if sessionID == 0 || roundNumber == 0 {
		http.Error(w, "missing session_id or round_number", 400)
		return
	}

	now := time.Now()
	res, err := db.Exec(`INSERT INTO session_rounds(session_id, round_number, started_at) VALUES(?,?,?)`, sessionID, roundNumber, now)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	roundID, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ok":true,"round_id":%d}`, roundID)
}

// handleRoundFinish 结束一轮训练
func handleRoundFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	roundID, _ := parseInt(r.FormValue("round_id"))
	if roundID == 0 {
		http.Error(w, "missing round_id", 400)
		return
	}

	success := r.FormValue("success") == "1" || r.FormValue("success") == "true"
	score, _ := parseInt(r.FormValue("score"))
	accuracy := 0.0
	if acc := r.FormValue("accuracy"); acc != "" {
		if f, err := strconv.ParseFloat(acc, 64); err == nil {
			accuracy = f
		}
	}
	metadata := strings.TrimSpace(r.FormValue("metadata"))

	// 优先使用前端传来的准确时长，如果没有则自动计算
	duration, _ := parseInt(r.FormValue("duration_seconds"))
	if duration == 0 {
		// 获取开始时间计算时长（回退方案）
		var startedAt time.Time
		if err := db.QueryRow(`SELECT started_at FROM session_rounds WHERE id=?`, roundID).Scan(&startedAt); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		duration = int(time.Now().Sub(startedAt).Seconds())
	}

	endedAt := time.Now()

	_, err := db.Exec(`UPDATE session_rounds SET ended_at=?, duration_seconds=?, success=?, score=?, accuracy=?, metadata=? WHERE id=?`,
		endedAt, duration, success, score, accuracy, metadata, roundID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// handleStatsDaily 获取每日统计数据
func handleStatsDaily(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 获取当日按训练项目聚合的统计数据（从session_rounds表）
	type TrainingStat struct {
		TrainingID   int     `json:"training_id"`
		TrainingName string  `json:"name"`
		TotalRounds  int     `json:"total_rounds"`
		TotalSeconds int     `json:"total_seconds"`
		AvgScore     float64 `json:"avg_score"`
		AvgAccuracy  float64 `json:"avg_accuracy"`
		SuccessRate  float64 `json:"success_rate"`
	}

	rows, err := db.Query(`
		SELECT
			t.id,
			t.name,
			COUNT(sr.id) as total_rounds,
			COALESCE(SUM(sr.duration_seconds), 0) as total_seconds,
			COALESCE(AVG(sr.score), 0) as avg_score,
			COALESCE(AVG(sr.accuracy), 0) as avg_accuracy,
			COALESCE(AVG(CASE WHEN sr.success = 1 THEN 100.0 ELSE 0.0 END), 0) as success_rate
		FROM trainings t
		JOIN sessions s ON t.id = s.training_id
		JOIN session_rounds sr ON s.id = sr.session_id
		WHERE substr(s.started_at, 1, 10) = ?
		GROUP BY t.id, t.name
		ORDER BY t.name
	`, date)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var trainings []TrainingStat
	totalDuration := 0
	totalRounds := 0
	for rows.Next() {
		var t TrainingStat
		if err := rows.Scan(&t.TrainingID, &t.TrainingName, &t.TotalRounds, &t.TotalSeconds,
			&t.AvgScore, &t.AvgAccuracy, &t.SuccessRate); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		totalDuration += t.TotalSeconds
		totalRounds += t.TotalRounds
		trainings = append(trainings, t)
	}

	w.Header().Set("Content-Type", "application/json")
	type Response struct {
		Date          string         `json:"date"`
		TotalDuration int            `json:"total_duration"`
		TotalMinutes  int            `json:"total_minutes"`
		TotalRounds   int            `json:"total_rounds"`
		Trainings     []TrainingStat `json:"trainings"`
	}
	resp := Response{
		Date:          date,
		TotalDuration: totalDuration,
		TotalMinutes:  totalDuration / 60,
		TotalRounds:   totalRounds,
		Trainings:     trainings,
	}

	// 简单的JSON编码
	jsonData, _ := json.Marshal(resp)
	w.Write(jsonData)
}

// handleStatsTraining 获取训练项目统计
func handleStatsTraining(w http.ResponseWriter, r *http.Request) {
	// 从URL路径提取training_id
	path := r.URL.Path
	idStr := strings.TrimPrefix(path, "/api/stats/training/")
	trainingID, _ := parseInt(idStr)
	if trainingID == 0 {
		http.Error(w, "missing training_id", 400)
		return
	}

	level := r.URL.Query().Get("level")
	timeRange := r.URL.Query().Get("range") // 新增：today 或 days
	days := 30

	// 计算开始日期
	var startDate string
	if timeRange == "today" {
		// 今天0点开始 - 只比较日期部分
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	} else {
		// 最近N天
		if d := r.URL.Query().Get("days"); d != "" {
			if n, err := strconv.Atoi(d); err == nil && n > 0 {
				days = n
			}
		}
		startDate = time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	}

	// 获取训练项目信息
	var trainingName string
	if err := db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&trainingName); err != nil {
		http.Error(w, "training not found", 404)
		return
	}

	query := `
		SELECT sr.round_number, sr.started_at, sr.duration_seconds, sr.success, sr.score, sr.accuracy, sr.metadata
		FROM session_rounds sr
		JOIN sessions s ON sr.session_id = s.id
		WHERE s.training_id = ? AND substr(sr.started_at, 1, 10) >= ?
	`
	args := []any{trainingID, startDate}

	// 只有明确指定level时才过滤，空字符串表示查询全部
	if level != "" && level != "all" {
		query += ` AND s.level = ?`
		args = append(args, level)
	}

	query += ` ORDER BY sr.started_at DESC LIMIT 100`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type RoundData struct {
		RoundNumber int     `json:"round_number"`
		StartedAt   string  `json:"started_at"`
		Duration    int     `json:"duration"`
		Success     bool    `json:"success"`
		Score       int     `json:"score"`
		Accuracy    float64 `json:"accuracy"`
		Metadata    string  `json:"metadata"`
	}

	var rounds []RoundData
	for rows.Next() {
		var r RoundData
		var success int
		var duration sql.NullInt64
		var score sql.NullInt64
		var accuracy sql.NullFloat64
		var metadata sql.NullString
		if err := rows.Scan(&r.RoundNumber, &r.StartedAt, &duration, &success, &score, &accuracy, &metadata); err != nil {
			log.Printf("扫描行出错: %v", err)
			continue
		}
		r.Success = success == 1
		r.Duration = int(duration.Int64)
		r.Score = int(score.Int64)
		r.Accuracy = accuracy.Float64
		if metadata.Valid {
			r.Metadata = metadata.String
		}
		rounds = append(rounds, r)
	}

	w.Header().Set("Content-Type", "application/json")
	type Response struct {
		TrainingID   int         `json:"training_id"`
		TrainingName string      `json:"training_name"`
		Level        string      `json:"level"`
		Days         int         `json:"days"`
		Rounds       []RoundData `json:"rounds"`
	}
	resp := Response{
		TrainingID:   trainingID,
		TrainingName: trainingName,
		Level:        level,
		Days:         days,
		Rounds:       rounds,
	}

	jsonData, _ := json.Marshal(resp)
	w.Write(jsonData)
}

// Template rendering
func renderTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
