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

// ToClashProxy converts the proxy to a Clash-compatible map with correct field names.
// VLESS Reality nodes need special handling:
//   - security: reality → tls: true + reality-opts:
//   - sni → servername
//   - fingerprint → client-fingerprint
//   - public-key, short-id → nested under reality-opts
func (p *Proxy) ToClashProxy() map[string]interface{} {
	m := map[string]interface{}{
		"name":   p.Name,
		"type":   p.Type,
		"server": p.Server,
		"port":   p.Port,
	}

	if p.Cipher != "" {
		m["cipher"] = p.Cipher
	}
	if p.Password != "" {
		m["password"] = p.Password
	}
	if p.UUID != "" {
		m["uuid"] = p.UUID
	}
	if p.Protocol != "" {
		m["protocol"] = p.Protocol
	}
	if p.Obfs != "" {
		m["obfs"] = p.Obfs
	}
	if p.ObfsParam != "" {
		m["obfs-param"] = p.ObfsParam
	}
	if p.Encryption != "" {
		m["encryption"] = p.Encryption
	}
	if p.Network != "" {
		m["network"] = p.Network
	}
	if p.Flow != "" {
		m["flow"] = p.Flow
	}
	if p.SkipCertVerify {
		m["skip-cert-verify"] = true
	}
	if p.Username != "" {
		m["username"] = p.Username
	}

	// VLESS Reality: use tls + reality-opts format
	if p.Security == "reality" {
		m["tls"] = true
		if p.SNI != "" {
			m["servername"] = p.SNI
		}
		if p.Fingerprint != "" {
			m["client-fingerprint"] = p.Fingerprint
		}
		// Reality options
		reality := map[string]interface{}{}
		if p.PublicKey != "" {
			reality["public-key"] = p.PublicKey
		}
		if p.ShortID != "" {
			reality["short-id"] = p.ShortID
		}
		if len(reality) > 0 {
			m["reality-opts"] = reality
		}
	} else {
		// Standard TLS
		if p.Security != "" {
			m["security"] = p.Security
		}
		if p.SNI != "" {
			m["sni"] = p.SNI
		}
		if p.Fingerprint != "" {
			m["fingerprint"] = p.Fingerprint
		}
		if p.PublicKey != "" {
			m["public-key"] = p.PublicKey
		}
		if p.ShortID != "" {
			m["short-id"] = p.ShortID
		}
	}

	// WebSocket specific
	if p.Network == "ws" {
		if p.WSPath != "" {
			m["ws-path"] = p.WSPath
		}
		if len(p.WSHeaders) > 0 {
			m["ws-headers"] = p.WSHeaders
		}
	}

	// Hysteria2 specific
	if p.ServerPorts != "" {
		m["server-ports"] = p.ServerPorts
	}
	if p.UpMbps > 0 {
		m["up-mbps"] = p.UpMbps
	}
	if p.DownMbps > 0 {
		m["down-mbps"] = p.DownMbps
	}
	if p.AuthStr != "" {
		m["auth-str"] = p.AuthStr
	}

	return m
}

// ProxyList is a sortable slice of proxies.
type ProxyList []Proxy

// Parser defines the interface for parsing different subscription formats.
type Parser interface {
	Parse(data []byte) (ProxyList, error)
}