import { useState, useEffect, useCallback } from "react";
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

interface SolutionData {
  scenario_id: string;
  name: string;
  description: string;
  fix_hint: string;
  hints: string[];
  difficulty: number;
  time_limit: number;
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

const HINT_INTERVAL = 60; // seconds between each hint reveal

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
  const [abandoning, setAbandoning] = useState(false);

  // End-of-game state (solution display)
  const [endState, setEndState] = useState<"completed" | "abandoned" | "timeout" | null>(null);
  const [solutionData, setSolutionData] = useState<SolutionData | null>(null);

  // Progressive hints: how many hints to reveal
  const hintsRevealed = timerElapsed != null && activeScenario
    ? Math.min(
        Math.floor(timerElapsed / HINT_INTERVAL) + 1,
        activeScenario.hints.length
      )
    : 0;

  // ── Fetch solution ────────────────────────────────────────────────────────

  const fetchSolution = useCallback(async (gameId: string) => {
    try {
      const res = await fetch(`/api/games/${gameId}/solution`);
      if (!res.ok) return;
      const data: SolutionData = await res.json();
      setSolutionData(data);
    } catch {
      // Ignore fetch errors
    }
  }, []);

  // ── Handle WS messages ────────────────────────────────────────────────────

  const handleMessage = (data: unknown) => {
    const msgStr = JSON.stringify(data);
    setLastMessage(msgStr);

    const evt = data as WsEvent;

    switch (evt.type) {
      case "game_started": {
        const scenario = evt.scenario as Scenario;
        const gameId = (evt.game as any)?.id ?? null;
        setActiveGameId(gameId);
        setActiveScenario(scenario);
        setTimerRemaining(scenario?.time_limit ?? null);
        setTimerElapsed(0);
        setAbandoning(false);
        setEndState(null);
        setSolutionData(null);
        break;
      }

      case "timer_tick": {
        const tick = evt as unknown as TimerTick;
        setTimerRemaining(tick.remaining);
        setTimerElapsed(tick.elapsed);
        break;
      }

      case "game_completed": {
        const gameId = (evt.game as any)?.id ?? activeGameId;
        setActiveGameId(null);
        setActiveScenario(null);
        setTimerRemaining(null);
        setTimerElapsed(null);
        setAbandoning(false);
        setEndState("completed");
        if (gameId) fetchSolution(gameId);
        break;
      }

      case "game_abandoned": {
        const gameId = (evt.game as any)?.id ?? activeGameId;
        setActiveGameId(null);
        setActiveScenario(null);
        setTimerRemaining(null);
        setTimerElapsed(null);
        setAbandoning(false);
        setEndState("abandoned");
        if (gameId) fetchSolution(gameId);
        break;
      }

      case "game_timeout": {
        const gameId = (evt.game as any)?.id ?? activeGameId;
        setActiveGameId(null);
        setActiveScenario(null);
        setTimerRemaining(null);
        setTimerElapsed(null);
        setAbandoning(false);
        setEndState("timeout");
        if (gameId) fetchSolution(gameId);
        break;
      }

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

  const { isConnected, sendMessage } = useWebSocket("/ws", handleMessage);

  useEffect(() => {
    setWsStatus(isConnected ? "connected" : "disconnected");
  }, [isConnected]);

  // ── Actions ────────────────────────────────────────────────────────────────

  const handleAbandon = useCallback(() => {
    if (!activeGameId || abandoning) return;
    setAbandoning(true);
    sendMessage(JSON.stringify({
      type: "abandon_game",
      game_id: activeGameId,
    }));
  }, [activeGameId, abandoning, sendMessage]);

  const handleDismissSolution = useCallback(() => {
    setEndState(null);
    setSolutionData(null);
  }, []);

  const affectedServiceId = activeScenario?.target_service_id ?? null;
  const affectedServiceName = affectedServiceId
    ? SERVICE_NAMES[affectedServiceId] ?? affectedServiceId
    : null;
  const affectedServiceColor = affectedServiceId
    ? SERVICE_COLORS[affectedServiceId] ?? "#FF4444"
    : "#FF4444";

  // ── End state label ────────────────────────────────────────────────────────

  const endStateLabel =
    endState === "completed" ? "🎉 INCIDENTE RESUELTO" :
    endState === "abandoned" ? "⏹ PARTIDA ABANDONADA" :
    endState === "timeout" ? "⏰ TIEMPO AGOTADO" : null;

  const endStateColor =
    endState === "completed" ? "#00FF7F" :
    endState === "abandoned" ? "#FF8C00" :
    endState === "timeout" ? "#FF4444" : "#fff";

  return (
    <div style={styles.container}>
      {/* ── Alarm Overlay (active game) ── */}
      {activeScenario && (
        <IncidentAlarm
          scenario={activeScenario}
          remaining={timerRemaining}
          elapsed={timerElapsed}
          serviceName={affectedServiceName ?? ""}
          serviceColor={affectedServiceColor}
          hintsRevealed={hintsRevealed}
          onAbandon={handleAbandon}
          abandoning={abandoning}
        />
      )}

      {/* ── Solution Overlay (game ended) ── */}
      {endState && solutionData && (
        <div style={styles.solutionOverlay}>
          <div style={styles.solutionPanel}>
            <div style={{ ...styles.solutionHeader, color: endStateColor }}>
              {endStateLabel}
            </div>

            <div style={styles.solutionScenarioName}>
              {solutionData.name}
            </div>
            <div style={styles.solutionDesc}>
              {solutionData.description}
            </div>

            <div style={styles.solutionDivider} />

            <div style={styles.solutionSection}>
              <div style={styles.solutionSectionTitle}>🔧 Solución</div>
              <div style={styles.solutionFix}>{solutionData.fix_hint}</div>
            </div>

            {solutionData.hints.length > 0 && (
              <div style={styles.solutionSection}>
                <div style={styles.solutionSectionTitle}>💡 Pistas disponibles</div>
                {solutionData.hints.map((h, i) => (
                  <div key={i} style={styles.solutionHint}>
                    {i + 1}. {h}
                  </div>
                ))}
              </div>
            )}

            <button
              style={styles.solutionDismissBtn}
              onClick={handleDismissSolution}
            >
              ✕ CERRAR
            </button>
          </div>
        </div>
      )}

      {/* ── Header ── */}
      <header style={styles.header}>
        <div style={styles.headerLeft}>
          <img
            src="/assets/logo-gidas.png"
            alt="GIDAS"
            style={styles.logoGidas}
          />
          <div>
            <h1 style={styles.title}>
              Downtime Game — Dashboard
              {activeGameId && (
                <span style={styles.liveBadge}>🔴 EN VIVO</span>
              )}
            </h1>
          </div>
        </div>
        <div style={styles.headerRight}>
          <img
            src="/assets/logo_infra_blanco.png"
            alt="INFRA IT"
            style={styles.logoInfra}
          />
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

      <footer style={styles.footer}>
        <div style={styles.footerContent}>
          <span style={styles.footerBrand}>
            <img
              src="/assets/logo-gidas.png"
              alt="GIDAS"
              style={styles.footerLogo}
            />
            <span style={styles.footerText}>
              Downtime Game — INFRA IT · GIDAS · UTN FRSF
            </span>
            <img
              src="/assets/logo-utn.svg"
              alt="UTN"
              style={styles.footerLogoUtn}
            />
          </span>
          {lastMessage && !activeScenario && !endState && (
            <code style={styles.footerWs}>WS: {lastMessage}</code>
          )}
        </div>
      </footer>
    </div>
  );
}

// ── Styles ────────────────────────────────────────────────────────────────────

const styles: Record<string, any> = {
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
    gap: "0.75rem",
  },
  headerLeft: {
    display: "flex",
    alignItems: "center",
    gap: "1rem",
  },
  headerRight: {
    display: "flex",
    alignItems: "center",
    gap: "1rem",
    flexWrap: "wrap",
  },
  logoGidas: {
    height: "36px",
    width: "auto",
    opacity: 0.9,
    filter: "brightness(1.2)",
  },
  logoInfra: {
    height: "32px",
    width: "auto",
    opacity: 0.8,
    filter: "brightness(1.1)",
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
    whiteSpace: "nowrap",
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
    padding: "0.75rem 1rem",
    background: "rgba(255,255,255,0.02)",
    borderTop: "1px solid rgba(255,255,255,0.06)",
  },
  footerContent: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    flexWrap: "wrap",
    gap: "0.5rem",
  },
  footerBrand: {
    display: "flex",
    alignItems: "center",
    gap: "0.75rem",
  },
  footerLogo: {
    height: "20px",
    width: "auto",
    opacity: 0.5,
  },
  footerLogoUtn: {
    height: "18px",
    width: "auto",
    opacity: 0.4,
  },
  footerText: {
    fontSize: "0.65rem",
    color: "rgba(255,255,255,0.25)",
    letterSpacing: "0.05rem",
  },
  footerWs: {
    fontSize: "0.65rem",
    color: "rgba(255,255,255,0.2)",
    letterSpacing: "0.05rem",
  },

  // ── Solution Overlay ──
  solutionOverlay: {
    position: "fixed",
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 2000,
    background: "rgba(0,0,0,0.85)",
    padding: "2rem",
  },
  solutionPanel: {
    background: "#12121a",
    border: "1px solid rgba(255,255,255,0.15)",
    borderRadius: "12px",
    padding: "2rem",
    maxWidth: "600px",
    width: "100%",
    maxHeight: "90vh",
    overflowY: "auto",
  },
  solutionHeader: {
    fontSize: "1.5rem",
    fontWeight: 900,
    textAlign: "center",
    marginBottom: "1rem",
    letterSpacing: "0.2rem",
  },
  solutionScenarioName: {
    fontSize: "1.2rem",
    fontWeight: 700,
    color: "#fff",
    textAlign: "center",
    marginBottom: "0.5rem",
  },
  solutionDesc: {
    fontSize: "0.85rem",
    color: "rgba(255,255,255,0.6)",
    textAlign: "center",
    lineHeight: 1.5,
    marginBottom: "1rem",
  },
  solutionDivider: {
    height: "1px",
    background: "rgba(255,255,255,0.1)",
    margin: "1rem 0",
  },
  solutionSection: {
    marginBottom: "1rem",
  },
  solutionSectionTitle: {
    fontSize: "0.9rem",
    color: "#FFD700",
    fontWeight: 700,
    marginBottom: "0.5rem",
    letterSpacing: "0.1rem",
  },
  solutionFix: {
    fontSize: "0.85rem",
    color: "#00FF7F",
    background: "rgba(0,255,127,0.08)",
    border: "1px solid rgba(0,255,127,0.2)",
    borderRadius: "6px",
    padding: "0.75rem 1rem",
    lineHeight: 1.5,
    fontFamily: "monospace",
  },
  solutionHint: {
    fontSize: "0.8rem",
    color: "rgba(255,255,255,0.7)",
    marginBottom: "0.35rem",
    lineHeight: 1.4,
    padding: "0.3rem 0.5rem",
    background: "rgba(255,255,255,0.03)",
    borderRadius: "4px",
  },
  solutionDismissBtn: {
    display: "block",
    margin: "1.5rem auto 0",
    padding: "0.6rem 2rem",
    background: "rgba(255,255,255,0.08)",
    border: "1px solid rgba(255,255,255,0.2)",
    borderRadius: "6px",
    color: "rgba(255,255,255,0.7)",
    fontFamily: "'Courier New', Courier, monospace",
    fontSize: "0.85rem",
    fontWeight: 700,
    cursor: "pointer",
    letterSpacing: "0.15rem",
    transition: "all 0.2s ease",
  },
};

export default App;
