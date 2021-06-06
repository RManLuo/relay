package rules

type Rule struct {
	Port   uint   `json:port`
	Remote string `json:remote`
	RIP    string
	Rport  uint   `json:rport`
	Type   string `json:type`
}
