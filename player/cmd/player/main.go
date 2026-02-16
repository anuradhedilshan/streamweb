package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type SessionState struct {
	Token string `json:"token"`
}

func statePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".streamweb-player.json")
}

func saveState(s SessionState) error {
	b, _ := json.Marshal(s)
	return os.WriteFile(statePath(), b, 0o600)
}

func loadState() (SessionState, error) {
	b, err := os.ReadFile(statePath())
	if err != nil {
		return SessionState{}, err
	}
	var s SessionState
	err = json.Unmarshal(b, &s)
	return s, err
}

func postJSON(url string, body any) (*http.Response, error) {
	b, _ := json.Marshal(body)
	return http.Post(url, "application/json", bytes.NewReader(b))
}

func login(api string) error {
	var email, password string
	fmt.Print("email: ")
	fmt.Scanln(&email)
	fmt.Print("password: ")
	fmt.Scanln(&password)

	res, err := postJSON(api+"/auth/login", map[string]string{"email": email, "password": password})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("login failed: %d", res.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	if out.AccessToken == "" {
		return fmt.Errorf("empty token")
	}
	if err := saveState(SessionState{Token: out.AccessToken}); err != nil {
		return err
	}
	fmt.Println("login success")
	return nil
}

func play(api, streamID string) error {
	s, err := loadState()
	if err != nil {
		return fmt.Errorf("not logged in")
	}
	res, err := postJSON(api+"/playback/start", map[string]string{"stream_id": streamID, "token": s.Token})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("playback start failed: %d", res.StatusCode)
	}
	var out struct {
		SessionID string `json:"session_id"`
		PlayURL   string `json:"play_url"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	fmt.Println("session:", out.SessionID)
	fmt.Println("play_url:", out.PlayURL)

	cmd := exec.Command("mpv", out.PlayURL)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mpv: %w", err)
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		hbRes, err := postJSON(api+"/playback/heartbeat", map[string]string{"session_id": out.SessionID})
		if err != nil {
			fmt.Println("heartbeat error:", err)
			continue
		}
		var hb struct {
			State   string `json:"state"`
			Balance int64  `json:"balance_points"`
		}
		_ = json.NewDecoder(hbRes.Body).Decode(&hb)
		hbRes.Body.Close()
		fmt.Printf("heartbeat state=%s balance=%d\n", hb.State, hb.Balance)
		if hb.State == "blocked" || hbRes.StatusCode == 402 {
			_ = cmd.Process.Kill()
			_, _ = postJSON(api+"/playback/stop", map[string]string{"session_id": out.SessionID})
			return fmt.Errorf("session blocked (points exhausted or kicked)")
		}
	}
	return nil
}

func logout() error {
	return os.Remove(statePath())
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: player <login|play|logout> [stream_id]")
		os.Exit(1)
	}
	api := os.Getenv("STREAMWEB_API")
	if api == "" {
		api = "http://127.0.0.1:8080"
	}

	var err error
	switch os.Args[1] {
	case "login":
		err = login(api)
	case "play":
		if len(os.Args) < 3 {
			fmt.Println("usage: player play <stream_id>")
			os.Exit(1)
		}
		err = play(api, os.Args[2])
	case "logout":
		err = logout()
	default:
		err = fmt.Errorf("unknown command")
	}
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}
