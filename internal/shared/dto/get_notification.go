package dto

import "ntx/internal/shared/rpc"

// GetNotificationMethod is the RPC method name to get one notification.
const GetNotificationMethod rpc.Method = "getNotification"

// GetNotificationParams has the parameters for GetNotificationMethod.
type GetNotificationParams struct {
	ID string `json:"id"`
}

// GetNotificationResult is the result object for GetNotificationMethod.
type GetNotificationResult struct {
	ID            string         `json:"id"`
	Timestamp     string         `json:"timestamp"`
	AppName       string         `json:"app_name"`
	ReplacesID    uint32         `json:"replaces_id"`
	Summary       string         `json:"summary"`
	Body          string         `json:"body"`
	Actions       []string       `json:"actions"`
	Hints         map[string]any `json:"hints"`
	ExpireTimeout int32          `json:"expire_timeout"`
	IconBase64    string         `json:"icon_base64"`
	IconMime      string         `json:"icon_mime"`
}
