package flow

type HistoryEvent struct {
	ID      int64
	Type    string
	Time    int64
	Payload map[string]any
}
