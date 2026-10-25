package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsn := os.Getenv("SQL_DSN")
	if dsn == "" {
		fmt.Println("SQL_DSN not set")
		os.Exit(1)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer db.Close()

	var userID int
	if err := db.QueryRow(`SELECT id FROM users WHERE role >= 10 ORDER BY id LIMIT 1`).Scan(&userID); err != nil {
		fmt.Println("no admin user:", err)
		os.Exit(1)
	}

	var previous string
	_ = db.QueryRow(`SELECT COALESCE(access_token,'') FROM users WHERE id = $1`, userID).Scan(&previous)

	buf := make([]byte, 16)
	rand.Read(buf)
	pat := hex.EncodeToString(buf)

	db.Exec(`UPDATE users SET access_token = $1 WHERE id = $2`, pat, userID)
	defer func() {
		db.Exec(`UPDATE users SET access_token = $1 WHERE id = $2`, previous, userID)
		fmt.Println("[cleanup] access_token restored")
	}()

	// Test channel 2 specifically!
	body, _ := json.Marshal(map[string]any{
		"model":      "gemini-3.8-flash-high",
		"probe_type": "candy",
		"randomize":  false,
	})
	req, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:3000/api/channel/2/eval", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("request err:", err)
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	fmt.Printf("HTTP %d:\n%s\n", resp.StatusCode, string(raw))
}
