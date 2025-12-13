package proposals

// Option configures a ProposalHandler.
type Option func(*ProposalHandler)

// WithRetainInjectedCommitInfo configures the ProposalHandler to pass the
// injected ExtendedCommitInfo through to the wrapped proposal handlers.
//
// By default, the handler removes the injected tx before calling wrapped
// handlers (and re-injects it after), to avoid confusing downstream logic that
// expects only SDK txs.
func WithRetainInjectedCommitInfo() Option {
	return func(p *ProposalHandler) {
		p.retainInjectedCommitInfoInWrappedHandler = true
	}
}
