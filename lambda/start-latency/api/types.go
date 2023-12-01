package api

type Event struct {
	CurTimeUsec int64  `json:"cur_time_micro"`
	SrvHTTP     string `json:"srv_http"`
}

type Result struct {
	ElapsedUsec int64 `json:"elapsed_usec"`
}
