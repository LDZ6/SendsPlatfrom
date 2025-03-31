package types

type ECardCertificate struct {
	JsSessionId string `json:"js_session_id"`
	HallTicket  string `json:"hall_ticket"`
}

type JwcCertificate struct {
	GsSession    string `json:"gs_session"`
	Emaphome_WEU string `json:"emaphome_WEU"`
}
