package ui

type Cfg struct {
	Version int
	Auth    authConfig `yaml:"Auth,flow"`
	TCP     tcpConfig  `yaml:"TCP,flow"`
	TLS     tlsConfig  `yaml:"TLS,flow"`
	HTTP    httpConfig `yaml:"HTTP,flow"`
	Logger  logConfig  `yaml:"Logger,flow"`
}

type authConfig struct {
	Users    []User   `yaml:"Users,flow"`
	Policies []Policy `yaml:"Policies,flow"`
}
type User struct {
	Username     string `yaml:"Username"`
	PasswordHash string `yaml:"PasswordHash"`
}
type Policy struct {
	Name      string   `yaml:"Name"`
	Allowance bool     `yaml:"Allowance"`
	Users     []string `yaml:"Users,flow"`
	Hosts     []string `yaml:"Hosts,flow"`
	Paths     []string `yaml:"Paths,flow"`
}

type ControllerConfig struct {
	AddressBindings []string `yaml:"AddressBindings"`

	Binds map[string][]string `yaml:"Binds,flow"`
}

type tcpProxierConfig struct {
	Routes []Route `yaml:"Routes,flow"`
}
type Route struct {
	Name     string `yaml:"Name"`
	Protocol string `yaml:"Protocol"`
	Backend  string `yaml:"Backend"`
}

type tcpConfig struct {
	Controller ControllerConfig `yaml:"Controller"`
	Proxier    tcpProxierConfig `yaml:"Proxier"`
}

type httpConfig struct {
	Midware MidwareConfig `yaml:"Midware"`
	Proxier ProxierConfig `yaml:"Proxier"`
}
type MidwareConfig struct {
	Binds []ServiceBind `yaml:"Binds,flow"`
}

type ServiceBind struct {
	Name  string   `yaml:"Name"`
	Id    string   `yaml:"Id"`
	Hosts []string `yaml:"Hosts"`
}
type ProxierConfig struct {
	Hosts []ProxyHost `yaml:"Hosts,flow"`
}
type ProxyHost struct {
	Name          string   `yaml:"Name"`
	Hosts         []string `yaml:"Hosts"`
	Backend       string   `yaml:"Backend"`
	TlsSkipVerify bool     `yaml:"TlsSkipVerify"`
}

type logConfig struct {
	UDP            UdpLoggerConfig `yaml:"UdpLogger"`
	EnableSSE      bool            `yaml:"EnableSSE"`
	File           string          `yaml:"File"`
	DisableConsole bool            `yaml:"DisableConsole"`
}

type tlsConfig struct {
	Certificates  []Certificate `yaml:"Certificates,flow"`
	MinTLSVersion int           `yaml:"MinTLSVersion"`
}

type Certificate struct {
	CertFile string `yaml:"CertFile"`
	KeyFile  string `yaml:"KeyFile"`
}
type UdpLoggerConfig struct {
	Address string `yaml:"Address"`
}