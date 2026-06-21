package internal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	qwenWebAuthBusinessScene  = "qwen_web"
	qwenWebAuthChatScene      = "qwen_chat"
	qwenWebAuthWorkspaceScene = "workspace_api"
	qwenWebSecurityBaseURL    = "https://sec.qianwen.com"
	qwenWebVersion            = "2.13.2"
	qwenWebFEVersion          = "1.0.0"
	qwenWebACSVersion         = "1.0.0"
)

type qwenWebACSToken struct {
	Dvidn     string
	Actkn     string
	Snver     string
	Bacsft    string
	UMIDToken string
	ExpiresAt time.Time
}

type qwenWebACSAuth struct {
	mu              sync.Mutex
	umidToken       string
	dvidn           string
	actkn           string
	snver           string
	bacsft          []string
	workspaceActkn  string
	workspaceBacsft []string
	expiresAt       time.Time
}

type queryPair struct {
	Key   string
	Value string
}

var loginACSAuth = &qwenWebACSAuth{}

func (a *qwenWebACSAuth) signedURLAndHeaders(account AccountRecord, path string, body []byte) (string, string, map[string]string, error) {
	return a.signedURLAndHeadersForScene(account, qwenChatAPIURL, path, nil, body, qwenWebAuthChatScene)
}

func (a *qwenWebACSAuth) signedURLAndHeadersForScene(account AccountRecord, baseURL, path string, extraPairs []queryPair, body []byte, scene string) (string, string, map[string]string, error) {
	token, err := a.consumeTokenForScene(scene)
	if err != nil {
		return "", "", nil, err
	}

	queryTS := time.Now().UnixMilli()
	reqt := time.Now().UnixMilli()
	ut := qwenWebUT(account)
	if ut == "" {
		ut = token.Dvidn
	}
	pairs := []queryPair{
		{Key: "biz_id", Value: "ai_qwen"},
		{Key: "fe_version", Value: qwenWebFEVersion},
		{Key: "chat_client", Value: "h5"},
		{Key: "device", Value: "pc"},
		{Key: "fr", Value: "pc"},
		{Key: "pr", Value: "qwen"},
		{Key: "ut", Value: ut},
		{Key: "la", Value: "zh-CN"},
		{Key: "tz", Value: "Asia/Shanghai"},
		{Key: "wv", Value: qwenWebVersion},
		{Key: "ve", Value: qwenWebVersion},
		{Key: "nonce", Value: generateRandomToken(11, "")},
		{Key: "timestamp", Value: fmt.Sprintf("%d", queryTS)},
	}
	pairs = append(pairs, extraPairs...)

	paramKeys := make([]string, 0, len(pairs))
	paramValue := strings.Builder{}
	rawQuery := strings.Builder{}
	for _, pair := range pairs {
		if pair.Value == "" {
			continue
		}
		if rawQuery.Len() > 0 {
			rawQuery.WriteString("&")
		}
		rawQuery.WriteString(url.QueryEscape(pair.Key))
		rawQuery.WriteString("=")
		rawQuery.WriteString(url.QueryEscape(pair.Value))
		paramKeys = append(paramKeys, pair.Key)
		paramValue.WriteString(pair.Value)
	}

	secret := fmt.Sprintf("%s:%d", token.Bacsft, reqt)
	bodySign := GenerateBodySignature(body, secret)
	sign := GenerateQwenWebSignature(token.Dvidn, qwenWebACSVersion, qwenWebKP(account), paramValue.String(), bodySign, token.Bacsft, reqt)
	headers := map[string]string{
		"clt-acs-sign":           sign,
		"clt-acs-reqt":           fmt.Sprintf("%d", reqt),
		"clt-acs-request-params": strings.Join(paramKeys, ","),
		"clt-acs-caer":           "vrad",
		"eo-clt-dvidn":           token.Dvidn,
		"eo-clt-sacsft":          token.Bacsft,
		"eo-clt-snver":           token.Snver,
		"eo-clt-actkn":           token.Actkn,
		"eo-clt-acs-ve":          qwenWebACSVersion,
		"eo-clt-acs-kp":          qwenWebKP(account),
	}
	if bodySign != "" {
		headers["clt-acs-bfg"] = bodySign
	}
	if token.UMIDToken != "" {
		headers["bx-umidtoken"] = token.UMIDToken
	}

	return strings.TrimRight(baseURL, "/") + path + "?" + rawQuery.String(), ut, headers, nil
}

func (a *qwenWebACSAuth) consumeToken() (*qwenWebACSToken, error) {
	return a.consumeTokenForScene(qwenWebAuthChatScene)
}

func (a *qwenWebACSAuth) consumeTokenForScene(scene string) (*qwenWebACSToken, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.ensureReadyLocked(); err != nil {
		return nil, err
	}
	if len(a.bacsft) == 0 {
		a.expiresAt = time.Time{}
		if err := a.ensureReadyLocked(); err != nil {
			return nil, err
		}
	}
	if len(a.bacsft) == 0 {
		return nil, fmt.Errorf("qianwen web ACS bacsft token is empty")
	}
	actkn := a.actkn
	bacsftTokens := &a.bacsft
	if scene == qwenWebAuthWorkspaceScene && len(a.workspaceBacsft) > 0 {
		actkn = defaultString(a.workspaceActkn, a.actkn)
		bacsftTokens = &a.workspaceBacsft
	}
	if len(*bacsftTokens) == 0 {
		return nil, fmt.Errorf("qianwen web ACS %s bacsft token is empty", scene)
	}
	bacsft := (*bacsftTokens)[0]
	*bacsftTokens = (*bacsftTokens)[1:]
	token := &qwenWebACSToken{
		Dvidn:     a.dvidn,
		Actkn:     actkn,
		Snver:     a.snver,
		Bacsft:    bacsft,
		UMIDToken: a.umidToken,
		ExpiresAt: a.expiresAt,
	}
	if len(*bacsftTokens) <= 5 {
		go a.refreshAsync()
	}
	return token, nil
}

func (a *qwenWebACSAuth) refreshAsync() {
	a.mu.Lock()
	defer a.mu.Unlock()
	_ = a.registerLocked(true)
}

func (a *qwenWebACSAuth) ensureReadyLocked() error {
	if a.dvidn != "" && a.actkn != "" && len(a.bacsft) > 0 && time.Now().Before(a.expiresAt.Add(-30*time.Minute)) {
		return nil
	}
	return a.registerLocked(false)
}

func (a *qwenWebACSAuth) registerLocked(forceUMID bool) error {
	if a.umidToken == "" || forceUMID {
		umid, err := GenerateUMIDTokenWithRetry(3)
		if err != nil {
			return fmt.Errorf("generate qianwen web UMID token: %w", err)
		}
		a.umidToken = umid
	}

	client, err := NewQianwenClient()
	if err != nil {
		return fmt.Errorf("create qianwen security client: %w", err)
	}
	resp, err := client.RegisterAndGetTokensForScenes(a.umidToken, qwenWebAuthBusinessScene, []string{qwenWebAuthChatScene, qwenWebAuthWorkspaceScene, "voice_command"}, qwenWebSecurityBaseURL)
	if err != nil {
		return fmt.Errorf("register qianwen web ACS token: %w", err)
	}

	a.dvidn = resp.Data.EoCltDvidn
	a.snver = resp.Data.EoCltSnver
	a.actkn = resp.Data.EoCltActkn
	a.bacsft = resp.Data.EoCltBacsft
	a.workspaceActkn = resp.Data.EoCltActkn
	a.workspaceBacsft = append([]string(nil), resp.Data.EoCltBacsft...)
	expiresAt := time.Unix(resp.Data.EoCltActknDl, 0)

	for _, relate := range resp.Data.UnifyRelate {
		switch relate.BusinessScene {
		case qwenWebAuthChatScene:
			if relate.EoCltActkn != "" {
				a.actkn = relate.EoCltActkn
			}
			if len(relate.EoCltBacsft) > 0 {
				a.bacsft = relate.EoCltBacsft
			}
			if relate.EoCltActknDl > 0 {
				expiresAt = time.Unix(relate.EoCltActknDl, 0)
			}
		case qwenWebAuthWorkspaceScene:
			if relate.EoCltActkn != "" {
				a.workspaceActkn = relate.EoCltActkn
			}
			if len(relate.EoCltBacsft) > 0 {
				a.workspaceBacsft = relate.EoCltBacsft
			}
		}
	}
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(2 * time.Hour)
	}
	a.expiresAt = expiresAt

	if a.dvidn == "" || a.actkn == "" || len(a.bacsft) == 0 {
		return fmt.Errorf("qianwen web ACS register returned incomplete token")
	}
	LogInfo("Qianwen Web ACS token ready: bacsft=%d expires=%s", len(a.bacsft), a.expiresAt.Format(time.RFC3339))
	return nil
}

func qwenWebUT(account AccountRecord) string {
	var localStorage map[string]interface{}
	if strings.TrimSpace(account.LocalStorageJSON) != "" {
		if err := json.Unmarshal([]byte(account.LocalStorageJSON), &localStorage); err == nil {
			for _, key := range []string{"uc-stat-dn", "b-user-id", "cna"} {
				if value, ok := localStorage[key]; ok {
					if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
						return text
					}
				}
			}
		}
	}
	for _, name := range []string{"b-user-id", "cna"} {
		if value := cookieValueFromHeader(account.CookieString, name); value != "" {
			return value
		}
	}
	return ""
}

func qwenWebKP(account AccountRecord) string {
	if value := cookieValueFromHeader(account.CookieString, "tongyi_sso_ticket_hash"); value != "" {
		return value
	}
	var cookies []qwenCookie
	if strings.TrimSpace(account.CookieJSON) != "" {
		if err := json.Unmarshal([]byte(account.CookieJSON), &cookies); err == nil {
			for _, cookie := range cookies {
				if cookie.Name == "tongyi_sso_ticket_hash" {
					return cookie.Value
				}
			}
		}
	}
	return ""
}
