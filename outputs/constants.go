package outputs

const (
	OK            string = "ok"
	Warning       string = "warning"
	Alert         string = "alert"
	Error         string = "error"
	Critical      string = "critical"
	Emergency     string = "emergency"
	Notice        string = "notice"
	Informational string = "informational"
	Debug         string = "debug"
	Info          string = "info"
	None          string = "none"

	All      string = "all"
	Fields   string = "fields"
	Total    string = "total"
	Rejected string = "rejected"
	Accepted string = "accepted"
	Outputs  string = "outputs"

	Rule      string = "rule"
	Priority  string = "priority"
	Source    string = "source"
	Tags      string = "tags"
	Time      string = "time"
	Text      string = "text"
	Plaintext string = "plaintext"
	JSON      string = "json"
	Markdown  string = "markdown"
	Hostname  string = "hostname"

	DefaultFooter  string = "https://github.com/kubearmor/KubeArmor"
	DefaultIconURL string = "https://github.com/kubearmor/KubeArmor/assets/47106543/2db0b636-5c82-49c0-bf7d-535e4ad0a991"

	// Colors
	PaleCyan  string = "#ccfff2"
	Yellow    string = "#ffc700"
	Red       string = "#e20b0b"
	LigthBlue string = "#68c2ff"
	Lightcyan string = "#5bffb5"
	Orange    string = "#ff5400"

	Kubeless  string = "Kubeless"
	Openfaas  string = "OpenFaas"
	Fission   string = "Fission"
	Kubearmor string = "Kubearmor"
	MQTT      string = "MQTT"

	UDP string = "udp"
	TCP string = "tcp"

	// SASL Auth mechanisms for SMTP
	Plain       string = "plain"
	OAuthBearer string = "oauthbearer"
	External    string = "external"
	Anonymous   string = "anonymous"
)
