package models

type SessionInfo struct {
	UserID    string `json:"user_id"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Location  string `json:"location"`
}

var SessionStore []SessionInfo
