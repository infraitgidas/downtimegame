import { useState, useEffect } from "react";
import ServiceCard from "./components/ServiceCard";
import IncidentAlarm from "./components/IncidentAlarm";
import useWebSocket from "./hooks/useWebSocket";

// ── Types ─────────────────────────────────────────────────────────────────────

interface Service {
  id: string;
  name: string;
  color: string;
  ip: string;
  port: number;
}

interface Scenario {
  id: string;
  name: string;
  description: string;
  target_service_id: string;
  failure_type: string;
  severity: string;
  difficulty: number;
  hints: string[];
  fix_hint: string;
  time_limit: number;
}

interface ServiceStatus {
  service_id: string;
  online: boolean;
  incident: boolean;
}

interface TimerTick {
  game_id: string;
  elapsed: number;
  remaining: number;
}

interface WsEvent {
  type: string;
  [key: string]: unknown;
}

// ── Constants ─────────────────────────────────────────────────────────────────

const DEFAULT_SERVICES: Service[] = [
  { id: "sg-rojo", name: "Rojo", color: "#DC143C", ip: "192.168.1.200", port: 8080 },
  { id: "sg-azul", name: "Azul", color: "#1E90FF", ip: "192.168.1.204", port: 8084 },
  { id: "sg-verde", name: "Verde", color: "#00FF7F", ip: "192.168.1.202", port: 8082 },
  { id: "sg-amarillo", name: "Amarillo", color: "#FFD700", ip: "192.168.1.203", port: 8083 },
];

const SERVICE_COLORS: Record<string, string> = {
  "sg-rojo": "#DC143C",
  "sg-azul": "#1E90FF",
  "sg-verde": "#00FF7F",
  "sg-amarillo": "#FFD700",
};

const SERVICE_NAMES: Record<string, string> = {
  "sg-rojo": "Rojo",
  "sg-azul": "Azul",
  "sg-verde": "Verde",
  "sg-amarillo": "Amarillo",
};

// ── App ──────────────────────────────────────────────────────────────────────

function App() {
  const [services] = useState<Service[]>(DEFAULT_SERVICES);
  const [wsStatus, setWsStatus] = useState<"connecting" | "connected" | "disconnected">("disconnected");
  const [serviceStatuses, setServiceStatuses] = useState<Record<string, ServiceStatus>>({});
  const [lastMessage, setLastMessage] = useState<string | null>(null);

  // Active game / alarm state
  const [activeGameId, setActiveGameId] = useState<string | null>(null);
  const [activeScenario, setActiveScenario] = useState<Scenario | null>(null);
  const [timerRemaining, setTimerRemaining] = useState<number | null>(null);
  const [timerElapsed, setTimerElapsed] = useState<number | null>(null);

  const handleMessage = (data: unknown) => {
    const msgStr = JSON.stringify(data);
    setLastMessage(msgStr);

    const evt = data as WsEvent;

    switch (evt.type) {
      case "game_started": {
        const scenario = evt.scenario as Scenario;
        setActiveGameId((evt.game as any)?.id ?? null);
        setActiveScenario(scenario);
        setTimerRemaining(scenario?.time_limit ?? null);
        setTimerElapsed(0);
        break;
      }

      case "timer_tick": {
        const tick = evt as unknown as TimerTick;
        setTimerRemaining(tick.remaining);
        setTimerElapsed(tick.elapsed);
        break;
      }

      case "game_completed":
      case "game_abandoned":
      case "game_timeout":
        setActiveGameId(null);
        setActiveScenario(null);
        setTimerRemaining(null);
        setTimerElapsed(null);
        break;

      case "incident_resolved":
        // Keep showing until game_completed arrives
        break;

      case "service_status": {
        const svcStatus = evt as unknown as ServiceStatus;
        setServiceStatuses((prev) => ({
          ...prev,
          [svcStatus.service_id]: svcStatus,
        }));
        break;
      }
    }
  };

  const { isConnected } = useWebSocket("/ws", handleMessage);

  useEffect(() => {
    setWsStatus(isConnected ? "connected" : "disconnected");
  }, [isConnected]);

  const affectedServiceId = activeScenario?.target_service_id ?? null;
  const affectedServiceName = affectedServiceId
    ? SERVICE_NAMES[affectedServiceId] ?? affectedServiceId
    : null;
  const affectedServiceColor = affectedServiceId
    ? SERVICE_COLORS[affectedServiceId] ?? "#FF4444"
    : "#FF4444";

  return (
    <div style={styles.container}>
      {/* ── Alarm Overlay ── */}
      {activeScenario && (
        <IncidentAlarm
          scenario={activeScenario}
          remaining={timerRemaining}
          elapsed={timerElapsed}
          serviceName={affectedServiceName ?? ""}
          serviceColor={affectedServiceColor}
        />
      )}

      {/* ── Header ── */}
      <header style={styles.header}>
        <h1 style={styles.title}>
          Downtime Game — Dashboard
          {activeGameId && (
            <span style={styles.liveBadge}>🔴 EN VIVO</span>
          )}
        </h1>
        <div style={styles.statusBar}>
          {activeScenario && affectedServiceName && (
            <span
              style={{
                ...styles.incidentBadge,
                borderColor: affectedServiceColor,
                color: affectedServiceColor,
                animation: "pulse-red 1s infinite",
              }}
            >
              🚨 {affectedServiceName}
            </span>
          )}
          <span style={styles.statusLabel}>WS:</span>
          <span
            style={{
              ...styles.statusDot,
              background: wsStatus === "connected" ? "#00FF7F" : "#FF4444",
            }}
          />
          <span style={styles.statusText}>{wsStatus}</span>
        </div>
      </header>

      {/* ── Service Grid ── */}
      <main style={styles.grid}>
        {services.map((svc) => {
          const svcStatus = serviceStatuses[svc.id];
          return (
            <ServiceCard
              key={svc.id}
              service={svc}
              hasIncident={svcStatus?.incident ?? false}
              isAffected={svc.id === affectedServiceId}
            />
          );
        })}
      </main>

      {lastMessage && !activeScenario && (
        <footer style={styles.footer}>
          <code>Last WS: {lastMessage}</code>
        </footer>
      )}
    </div>
  );
}

// ── Styles ────────────────────────────────────────────────────────────────────

const styles: Record<string, React.CSSProperties> = {
  container: {
    minHeight: "100vh",
    background: "#0a0a0f",
    color: "#ffffff",
    fontFamily: "'Courier New', Courier, monospace",
    padding: "2rem",
  },
  header: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: "2rem",
    borderBottom: "1px solid rgba(255,255,255,0.1)",
    paddingBottom: "1rem",
    flexWrap: "wrap",
    gap: "0.5rem",
  },
  title: {
    fontSize: "1.5rem",
    fontWeight: 700,
    color: "#00FF7F",
    textTransform: "uppercase",
    letterSpacing: "0.2rem",
    display: "flex",
    alignItems: "center",
    gap: "0.75rem",
  },
  liveBadge: {
    fontSize: "0.8rem",
    color: "#FF4444",
    animation: "pulse-red 1s infinite",
  },
  statusBar: {
    display: "flex",
    alignItems: "center",
    gap: "0.5rem",
    flexWrap: "wrap",
  },
  incidentBadge: {
    fontSize: "0.8rem",
    padding: "0.3rem 0.6rem",
    border: "1px solid",
    borderRadius: "4px",
    fontWeight: 700,
    letterSpacing: "0.1rem",
  },
  statusLabel: {
    fontSize: "0.8rem",
    color: "rgba(255,255,255,0.5)",
  },
  statusDot: {
    width: 10,
    height: 10,
    borderRadius: "50%",
    display: "inline-block",
  },
  statusText: {
    fontSize: "0.8rem",
    color: "rgba(255,255,255,0.7)",
    textTransform: "capitalize",
  },
  grid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
    gap: "1.5rem",
    maxWidth: "1400px",
    margin: "0 auto",
    position: "relative",
    zIndex: 1,
  },
  footer: {
    marginTop: "2rem",
    padding: "1rem",
    background: "rgba(255,255,255,0.03)",
    borderTop: "1px solid rgba(255,255,255,0.1)",
    textAlign: "center",
    fontSize: "0.75rem",
    color: "rgba(255,255,255,0.3)",
  },
};

export default App;
