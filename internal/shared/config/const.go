// Package config holds default values and env var names shared across ntx binaries.
package config

// Default paths and env var names used to configure ntx binaries.
const (
	DefaultSocketPath            = "/tmp/ntx.sock"
	DefaultDBPath                = ".local/share/ntx/ntx.db"
	SocketPathEnvName            = "NTX_SOCKET_PATH"
	DBPathEnvName                = "NTX_DB_PATH"
	ServerLogLevelEnvName        = "NTX_SERVER_LOG_LEVEL"
	DbusForwarderLogLevelEnvName = "NTX_DBUS_FORWARDER_LOG_LEVEL"
)
