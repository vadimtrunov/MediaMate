package qbittorrent

// qbitTorrent represents a torrent from the qBittorrent Web API.
type qbitTorrent struct {
	Hash       string  `json:"hash"`
	Name       string  `json:"name"`
	Size       int64   `json:"size"`
	Progress   float64 `json:"progress"`   // 0.0 to 1.0
	State      string  `json:"state"`      // "downloading", "uploading", "pausedDL", etc.
	DLSpeed    int64   `json:"dlspeed"`    // bytes/sec
	UPSpeed    int64   `json:"upspeed"`    // bytes/sec
	ETA        int64   `json:"eta"`        // seconds, 8640000 = infinity
	Downloaded int64   `json:"downloaded"` // bytes downloaded
	TotalSize  int64   `json:"total_size"` // total size in bytes
}
