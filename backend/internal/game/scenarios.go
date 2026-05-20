package game

// scenarios is the built-in catalog of downtime scenarios.
// Each scenario defines what breaks, which service is affected, and hints for the player.
var scenarios = []Scenario{
	{
		ID:              "svc-crash-rojo",
		Name:            "Servicio Rojo caído",
		Description:     "El servicio HTTP del contenedor Rojo dejó de responder. El proceso se detuvo inesperadamente.",
		TargetServiceID: "sg-rojo",
		FailureType:     "crash",
		Severity:        "critical",
		Difficulty:      2,
		Hints: []string{
			"Revisá si el servicio responde en el puerto 8080",
			"El proceso http-service.py puede haber terminado",
		},
		FixHint:   "Reiniciá el servicio con: systemctl restart sg-rojo",
		TimeLimit: 120,
	},
	{
		ID:              "svc-hang-azul",
		Name:            "Servicio Azul congelado",
		Description:     "El servicio Azul está corriendo pero no responde a las peticiones. Quedó en estado 'hang'.",
		TargetServiceID: "sg-azul",
		FailureType:     "hang",
		Severity:        "critical",
		Difficulty:      3,
		Hints: []string{
			"El proceso sigue activo pero no responde al health check",
			"Revisá los logs del servicio",
		},
		FixHint:   "Matá el proceso y reinicialo: systemctl restart sg-azul",
		TimeLimit: 150,
	},
	{
		ID:              "latency-verde",
		Name:            "Latencia en Servicio Verde",
		Description:     "El servicio Verde responde con latencia muy alta. Las peticiones tardan más de 5 segundos.",
		TargetServiceID: "sg-verde",
		FailureType:     "latency",
		Severity:        "major",
		Difficulty:      4,
		Hints: []string{
			"El health check responde pero muy lento",
			"Podría ser un proceso consumiendo CPU",
		},
		FixHint:   "Identificá el proceso que consume recursos y terminalo",
		TimeLimit: 180,
	},
	{
		ID:              "dns-failure-amarillo",
		Name:            "Fallo DNS en Servicio Amarillo",
		Description:     "El servicio Amarillo no puede resolver nombres de dominio internos. Las dependencias fallan.",
		TargetServiceID: "sg-amarillo",
		FailureType:     "dns_failure",
		Severity:        "major",
		Difficulty:      3,
		Hints: []string{
			"El servicio está online pero sus dependencias no funcionan",
			"Revisá la resolución DNS desde el contenedor",
		},
		FixHint:   "Verificá /etc/resolv.conf y reiniciá el servicio de resolución",
		TimeLimit: 150,
	},
	{
		ID:              "svc-crash-azul",
		Name:            "Servicio Azul detenido",
		Description:     "El servicio Azul fue detenido. No responde en el puerto 8084.",
		TargetServiceID: "sg-azul",
		FailureType:     "crash",
		Severity:        "critical",
		Difficulty:      2,
		Hints: []string{
			"El puerto 8084 no responde",
			"Revisá el estado del servicio con systemctl",
		},
		FixHint:   "Iniciá el servicio: systemctl start sg-azul",
		TimeLimit: 120,
	},
	{
		ID:              "latency-rojo",
			Name:            "Sobrecarga en Servicio Rojo",
			Description:     "El servicio Rojo está sobrecargado. Responde intermitentemente y con alta latencia.",
			TargetServiceID: "sg-rojo",
			FailureType:     "latency",
			Severity:        "major",
			Difficulty:      4,
			Hints: []string{
				"El servicio a veces responde y a veces no",
				"Podría ser un problema de recursos en el contenedor",
			},
			FixHint:   "Verificá el uso de CPU/memoria y reiniciá el servicio si es necesario",
			TimeLimit: 180,
		},
	{
		ID:              "svc-hang-verde",
		Name:            "Servicio Verde colgado",
		Description:     "El servicio Verde dejó de responder a pesar de que el proceso sigue ejecutándose.",
		TargetServiceID: "sg-verde",
		FailureType:     "hang",
		Severity:        "critical",
		Difficulty:      3,
		Hints: []string{
			"El proceso está vivo pero no procesa peticiones",
			"Revisá los file descriptors o conexiones abiertas",
		},
		FixHint:   "Forzá la terminación del proceso: kill -9 $(pgrep -f http-service.py) && systemctl restart sg-verde",
		TimeLimit: 150,
	},
	{
		ID:              "dns-failure-rojo",
		Name:            "DNS incorrecto en Rojo",
		Description:     "El Servicio Rojo tiene una configuración DNS incorrecta y no puede resolver nombres internos.",
		TargetServiceID: "sg-rojo",
		FailureType:     "dns_failure",
		Severity:        "major",
		Difficulty:      3,
		Hints: []string{
			"El servidor DNS configurado no es accesible",
			"Revisá /etc/resolv.conf",
		},
		FixHint:   "Configurá el DNS correcto: echo 'nameserver 192.168.1.253' > /etc/resolv.conf",
		TimeLimit: 120,
	},
}

// GetScenarios returns a copy of the scenario catalog.
func GetScenarios() []Scenario {
	result := make([]Scenario, len(scenarios))
	copy(result, scenarios)
	return result
}

// GetScenarioByID returns a scenario by its ID, or nil if not found.
func GetScenarioByID(id string) *Scenario {
	for _, s := range scenarios {
		if s.ID == id {
			return &s
		}
	}
	return nil
}

// GetScenarioForService returns a random scenario targeting the given service.
// For now returns the first matching one; randomization comes in the engine.
func GetScenarioForService(serviceID string) *Scenario {
	var matches []Scenario
	for _, s := range scenarios {
		if s.TargetServiceID == serviceID {
			matches = append(matches, s)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	// Return first match — engine will randomize
	return &matches[0]
}
