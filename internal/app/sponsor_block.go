package app

type SponsorBlockInfo struct {
}

type Segment struct {
	Id            string     `json:"UUID"`
	Segment       [2]float64 `json:"segment"`
	Category      string     `json:"category"`
	ActionType    string     `json:"actionType"`
	Locked        int        `json:"locked"`
	Votes         int        `json:"votes"`
	VideoDuration float64    `json:"videoDuration"`
	Description   string     `json:"description"`
}
