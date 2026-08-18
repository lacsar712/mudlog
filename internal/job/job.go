package job

import "time"

type Job struct {
	FrameID   string    `json:"frame_id"`
	RelayID   string    `json:"relay_id"`
	StoreID   string    `json:"store_id"`
	Type      string    `json:"type"`
	Body      []byte    `json:"body"`
	Attempt   int       `json:"attempt"`
	NotBefore time.Time `json:"not_before"`
	CreatedAt time.Time `json:"created_at"`
	ReplayOf  string    `json:"replay_of,omitempty"`
}

func (j Job) Clone() Job {
	c := j
	if j.Body != nil {
		c.Body = append([]byte(nil), j.Body...)
	}
	return c
}
