package types

type TotalRank struct {
	NickName string `json:"nick_name"`
	Score    int    `json:"score"`
}

type ScoreMessage struct {
	Points string `json:"points"`
	Detail string `json:"detail"`
}

type BroadcastMessage struct {
	Message    string `json:"message"`
	Ciphertext string `json:"ciphertext"`
	Token      string `json:"token"`
}
