package parser

// Proxy represents a single proxy node parsed from any format.
type Proxy struct {
	Name           string            `yaml:"name"`
	Type           string            `yaml:"type"`
	Server         string            `yaml:"server"`
	Port           int               `yaml:"port"`
	Cipher         string            `yaml:"cipher,omitempty"`
	Password       string            `yaml:"password,omitempty"`
	UUID           string            `yaml:"uuid,omitempty"`
	Protocol       string            `yaml:"protocol,omitempty"` // SSR protocol
	Obfs           string            `yaml:"obfs,omitempty"`
	ObfsParam      string            `yaml:"obfs-param,omitempty"`
	Flow           string            `yaml:"flow,omitempty"` // VLESS flow
	Encryption     string            `yaml:"encryption,omitempty"`
	SNI            string            `yaml:"sni,omitempty"`
	SkipCertVerify bool              `yaml:"skip-cert-verify,omitempty"`
	Network        string            `yaml:"network,omitempty"` // tcp, ws, grpc, h2
	WSPath         string            `yaml:"ws-path,omitempty"`
	WSHeaders      map[string]string `yaml:"ws-headers,omitempty"`
	Fingerprint    string            `yaml:"fingerprint,omitempty"`
	PublicKey      string            `yaml:"public-key,omitempty"` // Reality
	ShortID        string            `yaml:"short-id,omitempty"`   // Reality
	ServerPorts    string            `yaml:"server-ports,omitempty"` // Hysteria2
	UpMbps         int               `yaml:"up-mbps,omitempty"`
	DownMbps       int               `yaml:"down-mbps,omitempty"`
	AuthStr        string            `yaml:"auth-str,omitempty"` // Hysteria2 auth
	ProtocolParam  string            `yaml:"protocol-param,omitempty"` // SSR
	Security       string            `yaml:"security,omitempty"`  // tls, reality, etc.
	Username       string            `yaml:"username,omitempty"`  // socks/http auth
	
	Group          string            `yaml:"group,omitempty"` // Origin subscription
	Remarks        string            `yaml:"remarks,omitempty"`
}

// ProxyList is a sortable slice of proxies.
type ProxyList []Proxy

// Parser defines the interface for parsing different subscription formats.
type Parser interface {
	Parse(data []byte) (ProxyList, error)
}