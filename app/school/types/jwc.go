package types

type CookieInfo struct {
	Cookie string `json:"cookie"`
}

type ScheduleInfo struct {
	Gssession string `json:"gssession"`
	WEU       string `json:"weu"`
	Semester  string `json:"semester"`
}

type GpaInfo struct {
	Gssession string `json:"gssession"`
	WEU       string `json:"weu"`
}

type XueFenInfo struct {
	Gssession string `json:"gssession"`
	WEU       string `json:"weu"`
	Semester  string `json:"semester"`
}

type GradeInfo struct {
	Gssession string `json:"gssession"`
	WEU       string `json:"weu"`
	Semester  string `json:"semester"`
}

type JwcCertificate struct {
	GsSession    string `json:"gs_session"`
	Emaphome_WEU string `json:"emaphome_WEU"`
}
