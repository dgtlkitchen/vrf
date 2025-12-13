package types

type ABCIMethod string

const (
	PrepareProposal     ABCIMethod = "prepare_proposal"
	ProcessProposal     ABCIMethod = "process_proposal"
	ExtendVote          ABCIMethod = "extend_vote"
	VerifyVoteExtension ABCIMethod = "verify_vote_extension"
	PreBlock            ABCIMethod = "pre_blocker"
)

type MessageType string

const (
	MessageExtendedCommit MessageType = "extended_commit"
	MessageVoteExtension  MessageType = "vote_extension"
)
