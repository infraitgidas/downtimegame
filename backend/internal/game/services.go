package game

// ServiceConfig defines the known game services.
// These are the 4 LXC containers running HTTP services by color.
type ServiceConfig struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	IP    string `json:"ip"`
	Port  int    `json:"port"`
}

// DefaultServices returns the 4 game service configurations.
func DefaultServices() []ServiceConfig {
	return []ServiceConfig{
		{ID: "sg-rojo", Name: "Rojo", Color: "rojo", IP: "192.168.1.200", Port: 8080},
		{ID: "sg-azul", Name: "Azul", Color: "azul", IP: "192.168.1.204", Port: 8084},
		{ID: "sg-verde", Name: "Verde", Color: "verde", IP: "192.168.1.202", Port: 8082},
		{ID: "sg-amarillo", Name: "Amarillo", Color: "amarillo", IP: "192.168.1.203", Port: 8083},
	}
}

// ServiceHealthURL returns the full health check URL for a service.
func (s ServiceConfig) HealthURL() string {
	return "http://" + s.IP + ":" + itoa(s.Port) + "/health"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
