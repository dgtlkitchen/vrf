package types

const (
	// NumInjectedTxs is the number of injected transactions prepended to a
	// proposal when vote extensions are enabled and required by the application.
	NumInjectedTxs = 1

	// InjectedCommitInfoIndex is the index of the injected ExtendedCommitInfo in
	// the proposal's tx list.
	InjectedCommitInfoIndex = 0
)
