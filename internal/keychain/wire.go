package keychain

import "sync"

var (
	backendMu         sync.RWMutex
	backendFlagValue  string
	backendFlagWasSet bool
)

// SetBackendFlagOverride records the user-supplied --backend flag for
// the next openWith call. Called by root.WireBackendSelection at
// PersistentPreRunE time. flagSet matches cobra's pflag.Flag.Changed —
// true when the user passed --backend on the command line, regardless
// of whether the value is empty.
func SetBackendFlagOverride(value string, flagSet bool) {
	backendMu.Lock()
	defer backendMu.Unlock()
	backendFlagValue = value
	backendFlagWasSet = flagSet
}

// GetBackendFlagOverride returns the current override. Two return
// values preserve the flag-set vs unset distinction
// credstore.BindBackendFlag depends on.
func GetBackendFlagOverride() (value string, flagSet bool) {
	backendMu.RLock()
	defer backendMu.RUnlock()
	return backendFlagValue, backendFlagWasSet
}
