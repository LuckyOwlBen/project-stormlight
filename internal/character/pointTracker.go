package character

// PointTracker contains the common point allocation fields used across different character modules.
type PointTracker struct {
	TotalPoints     int  `json:"totalPoints"`
	PendingPoints   int  `json:"pendingPoints"`
	PointsRemaining int  `json:"pointsRemaining"`
	Finalized       bool `json:"finalized"`
}
