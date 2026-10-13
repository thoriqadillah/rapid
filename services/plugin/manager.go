// Package plugin hosts the external resolver plugin system (manifest scan,
// stdio JSON-RPC transport, first-match resolve chain). The full manager
// lands in S8; until then this stub only declares the type so
// services/download compiles with a nil-safe Plugins field.
package plugin

// Manager resolves URIs through external plugins. A nil *Manager means no
// plugins. Immutable after construction (S8), so no lock is needed.
type Manager struct{}
