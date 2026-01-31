package model

type RelationEvent struct {
	Type    string `json:"type"`
	UserID  int64  `json:"user_id"`
	PeerID  int64  `json:"peer_id"`
	Action  string `json:"action"`
	TraceID string `json:"trace_id,omitempty"`
}
