//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type loginChallengeResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ChallengeID string `json:"challenge_id"`
		MaskedPhone string `json:"masked_phone"`
	} `json:"data"`
}

type loginVerifyResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AccessToken        string  `json:"access_token"`
		RefreshToken       string  `json:"refresh_token"`
		MustChangePassword bool    `json:"must_change_password"`
		PasswordUpdatedAt  *string `json:"password_updated_at"`
	} `json:"data"`
}

type meResp struct {
	Code int `json:"code"`
	Data struct {
		User struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"user"`
		Permissions        []string `json:"permissions"`
		MustChangePassword bool     `json:"must_change_password"`
	} `json:"data"`
}

type refreshResp struct {
	Code int `json:"code"`
	Data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}

func postJSON(t *testing.T, url string, body map[string]any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http post: %v", err)
	}
	return resp
}

func postAuthJSON(t *testing.T, url, token string, body map[string]any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http post auth: %v", err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode resp: %v", err)
	}
}

func doJSONRequest(t *testing.T, h http.Handler, method, path, token string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeRecorder(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(out); err != nil {
		t.Fatalf("decode recorder: %v", err)
	}
}

func mustLoginChallenge(t *testing.T, doReq func(map[string]any) any, decodeFn func(any, any), account, password string) loginChallengeResp {
	t.Helper()
	raw := doReq(map[string]any{
		"account":  account,
		"password": password,
	})
	var out loginChallengeResp
	decodeFn(raw, &out)
	if out.Code != 0 || out.Data.ChallengeID == "" {
		t.Fatalf("login failed: %+v", out)
	}
	return out
}

func mustLoginVerify(t *testing.T, doReq func(map[string]any) any, decodeFn func(any, any), challengeID string) loginVerifyResp {
	t.Helper()
	raw := doReq(map[string]any{
		"challenge_id": challengeID,
		"otp_code":     "123456",
	})
	var out loginVerifyResp
	decodeFn(raw, &out)
	if out.Code != 0 || out.Data.AccessToken == "" {
		t.Fatalf("verify failed: %+v", out)
	}
	return out
}
