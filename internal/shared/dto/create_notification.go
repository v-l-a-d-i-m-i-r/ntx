// Package dto has the data transfer objects for the RPC calls between
// the dbus forwarder and the server.
package dto

import "ntx/internal/shared/rpc"

// CreateNotificationMethod is the RPC method name to create a notification.
const CreateNotificationMethod rpc.Method = "createNotification"

// CreateNotificationParams has the parameters for CreateNotificationMethod.
type CreateNotificationParams struct {
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

// CreateNotificationResult is the result object for
// CreateNotificationMethod.
type CreateNotificationResult struct {
	ID string `json:"id"`
}
