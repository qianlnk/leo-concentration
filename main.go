package main

import (
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
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
	"sync"
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

// 用户会话管理
type UserSession struct {
	UserID   int
	Username string
	Nickname string
	ExpireAt time.Time
}

var (
	sessionStore = make(map[string]*UserSession)
	sessionMutex sync.RWMutex
)

// generateSessionID 生成随机会话ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// getSession 从请求中获取用户会话
func getSession(r *http.Request) *UserSession {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()
	session, ok := sessionStore[cookie.Value]
	if !ok || session.ExpireAt.Before(time.Now()) {
		return nil
	}
	return session
}

// setSession 设置用户会话
func setSession(w http.ResponseWriter, userID int, username, nickname string) {
	sessionID := generateSessionID()
	sessionMutex.Lock()
	sessionStore[sessionID] = &UserSession{
		UserID:   userID,
		Username: username,
		Nickname: nickname,
		ExpireAt: time.Now().Add(7 * 24 * time.Hour), // 7天过期
	}
	sessionMutex.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7天
		HttpOnly: true,
	})
}

// clearSession 清除用户会话
func clearSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err == nil {
		sessionMutex.Lock()
		delete(sessionStore, cookie.Value)
		sessionMutex.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// requireLogin 检查登录状态的中间件
func requireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := getSession(r)
		if session == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

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

	// 登录相关路由（无需登录）
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/register", handleRegister)
	mux.HandleFunc("/logout", handleLogout)

	// 需要登录的路由
	mux.HandleFunc("/", requireLogin(handleIndex))
	mux.HandleFunc("/trainings", requireLogin(handleTrainings))
	mux.HandleFunc("/training", requireLogin(handleTraining))
	mux.HandleFunc("/session/start", requireLogin(handleSessionStart))
	mux.HandleFunc("/session", requireLogin(handleSession))
	mux.HandleFunc("/session/finish", requireLogin(handleSessionFinish))
	mux.HandleFunc("/reports/daily", requireLogin(handleReportsDaily))
	mux.HandleFunc("/reports/daily/v2", requireLogin(handleReportsDailyV2))
	mux.HandleFunc("/training/stats/", requireLogin(handleTrainingStats))
	mux.HandleFunc("/test/cup-ball", func(w http.ResponseWriter, r *http.Request) {
		renderTemplate(w, "test_cup_ball.html", nil)
	})
	mux.HandleFunc("/api/session/round/start", requireLogin(handleRoundStart))
	mux.HandleFunc("/api/session/round/finish", requireLogin(handleRoundFinish))
	mux.HandleFunc("/api/session/update-level", requireLogin(handleSessionUpdateLevel))
	mux.HandleFunc("/api/stats/daily", requireLogin(handleStatsDaily))
	mux.HandleFunc("/api/stats/training/", requireLogin(handleStatsTrainingRouter))
	mux.HandleFunc("/play/balance-hero", requireLogin(handlePlayBalance))
	mux.HandleFunc("/play/number-trace", requireLogin(handlePlayNumberTrace))
	mux.HandleFunc("/play/memory-cards", requireLogin(handlePlayMemory))
	mux.HandleFunc("/play/poem-trace", requireLogin(handlePlayPoemTrace))
	mux.HandleFunc("/play/warmup-jumping", requireLogin(handlePlayWarmup))
	mux.HandleFunc("/play/color-match", requireLogin(handlePlayColorMatch))
	mux.HandleFunc("/play/eagle-eye", requireLogin(handlePlayEagleEye))
	mux.HandleFunc("/play/cup-ball", requireLogin(handlePlayCupBall))
	mux.HandleFunc("/play/fish-adventure", requireLogin(handlePlayFishAdventure))
	mux.HandleFunc("/play/pattern-finder", requireLogin(handlePlayPatternFinder))

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

// handleLogin 处理登录页面和登录请求
func handleLogin(w http.ResponseWriter, r *http.Request) {
	// 如果已登录，跳转到首页
	if session := getSession(r); session != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		renderTemplate(w, "login.html", nil)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			renderTemplate(w, "login.html", map[string]any{"Error": "表单解析错误"})
			return
		}

		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		if username == "" || password == "" {
			renderTemplate(w, "login.html", map[string]any{"Error": "用户名和密码不能为空"})
			return
		}

		// 查询用户
		var userID int
		var storedPassword, nickname string
		err := db.QueryRow(`SELECT id, password, COALESCE(nickname, username) FROM users WHERE username = ?`, username).Scan(&userID, &storedPassword, &nickname)
		if err != nil {
			renderTemplate(w, "login.html", map[string]any{"Error": "用户名或密码错误"})
			return
		}

		// 简单密码验证（生产环境应使用bcrypt等加密）
		if password != storedPassword {
			renderTemplate(w, "login.html", map[string]any{"Error": "用户名或密码错误"})
			return
		}

		// 设置会话
		setSession(w, userID, username, nickname)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// handleRegister 处理注册页面和注册请求
func handleRegister(w http.ResponseWriter, r *http.Request) {
	// 如果已登录，跳转到首页
	if session := getSession(r); session != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		renderTemplate(w, "register.html", nil)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			renderTemplate(w, "register.html", map[string]any{"Error": "表单解析错误"})
			return
		}

		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")
		nickname := strings.TrimSpace(r.FormValue("nickname"))

		if username == "" || password == "" {
			renderTemplate(w, "register.html", map[string]any{"Error": "用户名和密码不能为空"})
			return
		}

		if len(username) < 3 {
			renderTemplate(w, "register.html", map[string]any{"Error": "用户名至少3个字符"})
			return
		}

		if len(password) < 4 {
			renderTemplate(w, "register.html", map[string]any{"Error": "密码至少4个字符"})
			return
		}

		if password != confirmPassword {
			renderTemplate(w, "register.html", map[string]any{"Error": "两次密码输入不一致"})
			return
		}

		if nickname == "" {
			nickname = username
		}

		// 检查用户名是否已存在
		var exists int
		db.QueryRow(`SELECT COUNT(1) FROM users WHERE username = ?`, username).Scan(&exists)
		if exists > 0 {
			renderTemplate(w, "register.html", map[string]any{"Error": "用户名已存在"})
			return
		}

		// 创建用户（简单存储密码，生产环境应使用bcrypt）
		result, err := db.Exec(`INSERT INTO users (username, password, nickname) VALUES (?, ?, ?)`, username, password, nickname)
		if err != nil {
			renderTemplate(w, "register.html", map[string]any{"Error": "注册失败，请重试"})
			return
		}

		userID, _ := result.LastInsertId()

		// 自动登录
		setSession(w, int(userID), username, nickname)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// handleLogout 处理登出请求
func handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL UNIQUE,
            password TEXT NOT NULL,
            nickname TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
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
            user_id INTEGER,
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
            FOREIGN KEY(user_id) REFERENCES users(id),
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

	addColumnIfNotExists("sessions", "user_id", "INTEGER")
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
	if err := ensure("连珠大师", "pattern-finder", "在15×15棋盘上找出所有4个相连的黑白图案，训练视觉搜索与模式识别能力。", "观察/专注", 10); err != nil {
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

	// 获取当前用户
	userSession := getSession(r)
	if userSession == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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

	// 使用LEFT JOIN查询所有训练项目及其今天的统计数据（只查询当前用户的数据）
	query := `
		SELECT
			t.id,
			t.name,
			t.category,
			t.suggested_minutes,
			COALESCE(SUM(sr.duration_seconds), 0) as today_seconds,
			COALESCE(COUNT(sr.id), 0) as today_rounds
		FROM trainings t
		LEFT JOIN sessions s ON t.id = s.training_id AND substr(s.started_at, 1, 10) = ? AND s.user_id = ?
		LEFT JOIN session_rounds sr ON s.id = sr.session_id
		WHERE t.active = 1
		GROUP BY t.id, t.name, t.category, t.suggested_minutes
		ORDER BY t.id
	`

	rows, err := db.Query(query, today, userSession.UserID)
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
		"User":      userSession,
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

	// 获取当前用户ID
	session := getSession(r)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	now := time.Now()
	res, err := db.Exec(`INSERT INTO sessions(user_id, training_id, started_at) VALUES(?,?,?)`, session.UserID, tid, now)
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

// handlePlayPatternFinder 处理连珠大师游戏页面请求
func handlePlayPatternFinder(w http.ResponseWriter, r *http.Request) {
	id, _ := parseInt(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", 400)
		return
	}
	var trainingID int
	_ = db.QueryRow(`SELECT training_id FROM sessions WHERE id=?`, id).Scan(&trainingID)
	var name string
	_ = db.QueryRow(`SELECT name FROM trainings WHERE id=?`, trainingID).Scan(&name)
	renderTemplate(w, "play_pattern_finder.html", map[string]any{"ID": id, "TrainingName": name})
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

// handleSessionUpdateLevel 更新会话的等级信息
func handleSessionUpdateLevel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	sessionID, _ := parseInt(r.FormValue("session_id"))
	level := strings.TrimSpace(r.FormValue("level"))
	if sessionID == 0 {
		http.Error(w, "missing session_id", 400)
		return
	}
	if level == "" {
		http.Error(w, "missing level", 400)
		return
	}

	_, err := db.Exec(`UPDATE sessions SET level = ? WHERE id = ?`, level, sessionID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
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
	// 获取当前用户
	userSession := getSession(r)
	if userSession == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 获取当日按训练项目聚合的统计数据（从session_rounds表，只查询当前用户）
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
		WHERE substr(s.started_at, 1, 10) = ? AND s.user_id = ?
		GROUP BY t.id, t.name
		ORDER BY t.name
	`, date, userSession.UserID)
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

// handleStatsTrainingRouter 路由分发：根据路径决定调用哪个处理函数
func handleStatsTrainingRouter(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/levels") {
		handleStatsTrainingLevels(w, r)
	} else {
		handleStatsTraining(w, r)
	}
}

// handleStatsTraining 获取训练项目统计
func handleStatsTraining(w http.ResponseWriter, r *http.Request) {
	// 获取当前用户
	userSession := getSession(r)
	if userSession == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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
		SELECT sr.round_number, sr.started_at, sr.duration_seconds, sr.success, sr.score, sr.accuracy, sr.metadata, COALESCE(s.level, '')
		FROM session_rounds sr
		JOIN sessions s ON sr.session_id = s.id
		WHERE s.training_id = ? AND substr(sr.started_at, 1, 10) >= ? AND s.user_id = ?
	`
	args := []any{trainingID, startDate, userSession.UserID}

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
		Level       string  `json:"level"`
	}

	var rounds []RoundData
	for rows.Next() {
		var r RoundData
		var success int
		var duration sql.NullInt64
		var score sql.NullInt64
		var accuracy sql.NullFloat64
		var metadata sql.NullString
		if err := rows.Scan(&r.RoundNumber, &r.StartedAt, &duration, &success, &score, &accuracy, &metadata, &r.Level); err != nil {
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

// handleStatsTrainingLevels 获取训练项目的所有等级列表
func handleStatsTrainingLevels(w http.ResponseWriter, r *http.Request) {
	// 获取当前用户
	userSession := getSession(r)
	if userSession == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 从URL路径提取training_id
	path := r.URL.Path
	idStr := strings.TrimPrefix(path, "/api/stats/training/")
	idStr = strings.TrimSuffix(idStr, "/levels")
	trainingID, _ := parseInt(idStr)
	if trainingID == 0 {
		http.Error(w, "missing training_id", 400)
		return
	}

	// 获取该训练项目的所有不同等级（只查询当前用户）
	rows, err := db.Query(`
		SELECT DISTINCT COALESCE(level, '') as level
		FROM sessions
		WHERE training_id = ? AND user_id = ? AND level IS NOT NULL AND level != ''
		ORDER BY level
	`, trainingID, userSession.UserID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var levels []string
	for rows.Next() {
		var level string
		if err := rows.Scan(&level); err != nil {
			continue
		}
		if level != "" {
			levels = append(levels, level)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	jsonData, _ := json.Marshal(map[string]any{
		"training_id": trainingID,
		"levels":      levels,
	})
	w.Write(jsonData)
}

// Template rendering
func renderTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
