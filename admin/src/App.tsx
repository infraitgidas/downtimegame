import { useState, useEffect, useCallback } from "react";
import useWebSocket from "./useWebSocket";

// ── Types ─────────────────────────────────────────────────────────────────────

interface Game {
  id: string;
  player_name: string;
  status: string;
  score: number;
  started_at: string | null;
  ended_at: string | null;
  created_at: string;
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

interface LeaderboardEntry {
  player_name: string;
  score: number;
  time_seconds: number;
  played_at: string;
}

interface WsEvent {
  type: string;
  [key: string]: unknown;
}

// ── Constants ─────────────────────────────────────────────────────────────────

const SERVICE_NAMES: Record<string, string> = {
  "sg-rojo": "Rojo",
  "sg-azul": "Azul",
  "sg-verde": "Verde",
  "sg-amarillo": "Amarillo",
};

const STATUS_COLORS: Record<string, string> = {
  pending: "#FFD700",
  active: "#FF4444",
  completed: "#00FF7F",
  abandoned: "#666",
  timeout: "#FF8C00",
};

// ── API helpers ──────────────────────────────────────────────────────────────

async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(path);
  if (!res.ok) throw new Error(`GET ${path}: ${res.status}`);
  return res.json();
}

async function apiPost<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const errBody = await res.text();
    throw new Error(`POST ${path}: ${res.status} — ${errBody}`);
  }
  return res.json();
}

// ── App ──────────────────────────────────────────────────────────────────────

function App() {
  // Data
  const [games, setGames] = useState<Game[]>([]);
  const [scenarios, setScenarios] = useState<Scenario[]>([]);
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);

  // Game controls
  const [playerName, setPlayerName] = useState("");
  const [pendingGames, setPendingGames] = useState<Game[]>([]);
  const [selectedGameId, setSelectedGameId] = useState("");
  const [activeGameId, setActiveGameId] = useState<string | null>(null);
  const [activeScenario, setActiveScenario] = useState<Scenario | null>(null);
  const [actionMsg, setActionMsg] = useState<string | null>(null);

  // Logs
  const [logs, setLogs] = useState<string[]>([]);

  // Fetch data on mount
  useEffect(() => {
    apiGet<Game[]>("/api/games").then(setGames).catch(console.error);
    apiGet<Scenario[]>("/api/scenarios").then(setScenarios).catch(console.error);
    apiGet<LeaderboardEntry[]>("/api/leaderboard").then(setLeaderboard).catch(console.error);
  }, []);

  // Update pending games list when games change
  useEffect(() => {
    setPendingGames(games.filter((g) => g.status === "pending"));
  }, [games]);

  // WS handler
  const handleWsMessage = (data: unknown) => {
    const msg = typeof data === "string" ? data : JSON.stringify(data);
    setLogs((prev) => [msg, ...prev].slice(0, 100));

    const evt = data as WsEvent;
    const now = new Date().toLocaleTimeString();

    switch (evt.type) {
      case "game_created":
        // Refresh games list
        apiGet<Game[]>("/api/games").then(setGames).catch(console.error);
        setActionMsg(`${now} — Partida creada`);
        break;

      case "game_started": {
        setActiveGameId((evt.game as Game)?.id ?? null);
        setActiveScenario(evt.scenario as Scenario ?? null);
        setActionMsg(`${now} — Partida INICIADA`);
        apiGet<Game[]>("/api/games").then(setGames).catch(console.error);
        break;
      }

      case "game_completed":
        setActiveGameId(null);
        setActiveScenario(null);
        setActionMsg(`${now} — Partida COMPLETADA (score: ${evt.score})`);
        apiGet<Game[]>("/api/games").then(setGames).catch(console.error);
        apiGet<LeaderboardEntry[]>("/api/leaderboard").then(setLeaderboard).catch(console.error);
        break;

      case "game_abandoned":
      case "game_timeout":
        setActiveGameId(null);
        setActiveScenario(null);
        setActionMsg(`${now} — Partida ${evt.type === "game_abandoned" ? "ABANDONADA" : "TIMEOUT"}`);
        apiGet<Game[]>("/api/games").then(setGames).catch(console.error);
        break;

      case "incident_resolved":
        setActionMsg(`${now} — Incidente resuelto (score: ${evt.score})`);
        break;
    }
  };

  const { isConnected } = useWebSocket("/ws", handleWsMessage);

  // ── Actions ────────────────────────────────────────────────────────────────

  const handleCreateGame = useCallback(async () => {
    if (!playerName.trim()) return;
    try {
      await apiPost("/api/games", { player_name: playerName.trim() });
      setPlayerName("");
    } catch (err) {
      setActionMsg(`Error: ${err}`);
    }
  }, [playerName]);

  const handleStartGame = useCallback(async () => {
    if (!selectedGameId) return;
    try {
      await apiPost(`/api/games/${selectedGameId}/start`);
    } catch (err) {
      setActionMsg(`Error: ${err}`);
    }
  }, [selectedGameId]);

  const handleResolveIncident = useCallback(async () => {
    if (!activeGameId) return;
    try {
      await apiPost(`/api/games/${activeGameId}/resolve`);
    } catch (err) {
      setActionMsg(`Error: ${err}`);
    }
  }, [activeGameId]);

  const handleAbandonGame = useCallback(async () => {
    if (!activeGameId) return;
    try {
      await apiPost(`/api/games/${activeGameId}/abandon`);
    } catch (err) {
      setActionMsg(`Error: ${err}`);
    }
  }, [activeGameId]);

  const formatDate = (s: string | null) => {
    if (!s) return "—";
    return new Date(s).toLocaleString();
  };

  return (
    <div style={styles.container}>
      <header style={styles.header}>
        <h1 style={styles.title}>Downtime Game — Admin</h1>
        <div style={styles.headerRight}>
          {actionMsg && <span style={styles.actionMsg}>{actionMsg}</span>}
          <div style={styles.statusBar}>
            <span
              style={{
                ...styles.dot,
                background: isConnected ? "#00FF7F" : "#FF4444",
              }}
            />
            <span style={styles.statusText}>
              {isConnected ? "Conectado" : "Desconectado"}
            </span>
          </div>
        </div>
      </header>

      <div style={styles.grid}>
        {/* ── Game Controls ── */}
        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>Controles de Juego</h2>

          {/* Create Game */}
          <div style={styles.controlRow}>
            <input
              style={styles.input}
              placeholder="Nombre del jugador"
              value={playerName}
              onChange={(e) => setPlayerName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleCreateGame()}
            />
            <button
              style={styles.btn}
              onClick={handleCreateGame}
              disabled={!playerName.trim()}
            >
              + Crear Partida
            </button>
          </div>

          {/* Start Game */}
          <div style={styles.controlRow}>
            <select
              style={styles.select}
              value={selectedGameId}
              onChange={(e) => setSelectedGameId(e.target.value)}
            >
              <option value="">-- Seleccionar partida pendiente --</option>
              {pendingGames.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.id.slice(0, 8)} — {g.player_name}
                </option>
              ))}
            </select>
            <button
              style={styles.btn}
              onClick={handleStartGame}
              disabled={!selectedGameId}
            >
              ▶ Iniciar Partida
            </button>
          </div>

          {/* Active game controls */}
          {activeGameId && (
            <div style={styles.activeBanner}>
              <div style={styles.activeInfo}>
                <span style={styles.activeLabel}>Partida activa:</span>
                <code style={styles.activeId}>{activeGameId.slice(0, 8)}</code>
                {activeScenario && (
                  <>
                    <span style={styles.badgeSeverity(activeScenario.severity)}>
                      {activeScenario.failure_type.toUpperCase()}
                    </span>
                    <span style={styles.targetService}>
                      🎯 {SERVICE_NAMES[activeScenario.target_service_id] ?? activeScenario.target_service_id}
                    </span>
                  </>
                )}
              </div>
              <div style={styles.activeButtons}>
                <button style={styles.btnResolve} onClick={handleResolveIncident}>
                  ✅ Resolver Incidente
                </button>
                <button style={styles.btnAbandon} onClick={handleAbandonGame}>
                  ⏹ Abandonar Partida
                </button>
              </div>
            </div>
          )}
        </section>

        {/* ── Games List ── */}
        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>Partidas</h2>
          <div style={styles.tableWrap}>
            <table style={styles.table}>
              <thead>
                <tr>
                  <th style={styles.th}>ID</th>
                  <th style={styles.th}>Jugador</th>
                  <th style={styles.th}>Estado</th>
                  <th style={styles.th}>Puntaje</th>
                  <th style={styles.th}>Inicio</th>
                  <th style={styles.th}>Fin</th>
                </tr>
              </thead>
              <tbody>
                {games.length === 0 ? (
                  <tr>
                    <td style={styles.td} colSpan={6}>
                      Sin partidas aún
                    </td>
                  </tr>
                ) : (
                  games.map((g) => (
                    <tr key={g.id}>
                      <td style={styles.td}>
                        <code>{g.id.slice(0, 8)}</code>
                      </td>
                      <td style={styles.td}>{g.player_name}</td>
                      <td style={styles.td}>
                        <span
                          style={{
                            ...styles.statusBadge,
                            color: STATUS_COLORS[g.status] ?? "#fff",
                            borderColor: STATUS_COLORS[g.status] ?? "#fff",
                          }}
                        >
                          {g.status.toUpperCase()}
                        </span>
                      </td>
                      <td style={styles.td}>
                        {g.score > 0 ? g.score.toLocaleString() : "—"}
                      </td>
                      <td style={styles.td}>{formatDate(g.started_at)}</td>
                      <td style={styles.td}>{formatDate(g.ended_at)}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* ── Leaderboard ── */}
        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>Leaderboard</h2>
          <div style={styles.tableWrap}>
            {leaderboard.length === 0 ? (
              <p style={styles.empty}>Sin puntajes aún</p>
            ) : (
              <table style={styles.table}>
                <thead>
                  <tr>
                    <th style={styles.th}>#</th>
                    <th style={styles.th}>Jugador</th>
                    <th style={styles.th}>Puntaje</th>
                    <th style={styles.th}>Tiempo</th>
                    <th style={styles.th}>Fecha</th>
                  </tr>
                </thead>
                <tbody>
                  {leaderboard.map((e, i) => (
                    <tr key={i}>
                      <td style={{ ...styles.td, color: "#FFD700" }}>#{i + 1}</td>
                      <td style={styles.td}>{e.player_name}</td>
                      <td style={{ ...styles.td, color: "#00FF7F", fontWeight: 700 }}>
                        {e.score.toLocaleString()}
                      </td>
                      <td style={styles.td}>{e.time_seconds}s</td>
                      <td style={styles.td}>
                        {new Date(e.played_at).toLocaleDateString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </section>

        {/* ── Scenarios ── */}
        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>Escenarios ({scenarios.length})</h2>
          <div style={styles.scenarioGrid}>
            {scenarios.map((s) => (
              <div key={s.id} style={styles.scenarioCard}>
                <div style={styles.scenarioHeader}>
                  <span style={styles.scenarioName}>{s.name}</span>
                  <span style={styles.badgeSeverity(s.severity)}>
                    {s.failure_type.toUpperCase()}
                  </span>
                </div>
                <div style={styles.scenarioMeta}>
                  <span>🎯 {SERVICE_NAMES[s.target_service_id] ?? s.target_service_id}</span>
                  <span>⭐ Dificultad {s.difficulty}/5</span>
                  <span>⏱ {s.time_limit}s</span>
                </div>
                <p style={styles.scenarioDesc}>{s.description}</p>
              </div>
            ))}
          </div>
        </section>

        {/* ── WS Logs ── */}
        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>WebSocket Events</h2>
          <div style={styles.logContainer}>
            {logs.length === 0 ? (
              <p style={styles.logEmpty}>Esperando mensajes...</p>
            ) : (
              logs.map((log, i) => (
                <code key={i} style={styles.logLine}>
                  {log}
                </code>
              ))
            )}
          </div>
        </section>
      </div>
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
    padding: "1rem 2rem",
  },
  header: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: "1.5rem",
    borderBottom: "1px solid rgba(255,255,255,0.1)",
    paddingBottom: "1rem",
    flexWrap: "wrap",
    gap: "0.5rem",
  },
  title: {
    fontSize: "1.5rem",
    fontWeight: 700,
    color: "#FFD700",
    textTransform: "uppercase",
    letterSpacing: "0.2rem",
  },
  headerRight: {
    display: "flex",
    alignItems: "center",
    gap: "1rem",
    flexWrap: "wrap",
  },
  actionMsg: {
    fontSize: "0.8rem",
    color: "#00FF7F",
    background: "rgba(0,255,127,0.1)",
    padding: "0.3rem 0.6rem",
    borderRadius: "4px",
    border: "1px solid rgba(0,255,127,0.2)",
  },
  statusBar: {
    display: "flex",
    alignItems: "center",
    gap: "0.5rem",
  },
  dot: {
    width: 10,
    height: 10,
    borderRadius: "50%",
    display: "inline-block",
  },
  statusText: {
    fontSize: "0.8rem",
    color: "rgba(255,255,255,0.7)",
  },
  grid: {
    display: "flex",
    flexDirection: "column",
    gap: "1.5rem",
  },
  section: {
    background: "rgba(255,255,255,0.02)",
    border: "1px solid rgba(255,255,255,0.08)",
    borderRadius: "8px",
    padding: "1.5rem",
  },
  sectionTitle: {
    fontSize: "0.9rem",
    color: "#FFD700",
    marginBottom: "1rem",
    textTransform: "uppercase",
    letterSpacing: "0.15rem",
  },
  controlRow: {
    display: "flex",
    gap: "0.5rem",
    marginBottom: "0.75rem",
    flexWrap: "wrap",
  },
  input: {
    flex: 1,
    minWidth: "200px",
    padding: "0.6rem 0.8rem",
    background: "rgba(0,0,0,0.4)",
    border: "1px solid rgba(255,255,255,0.15)",
    borderRadius: "4px",
    color: "#fff",
    fontFamily: "monospace",
    fontSize: "0.85rem",
    outline: "none",
  },
  select: {
    flex: 1,
    minWidth: "280px",
    padding: "0.6rem 0.8rem",
    background: "rgba(0,0,0,0.4)",
    border: "1px solid rgba(255,255,255,0.15)",
    borderRadius: "4px",
    color: "#fff",
    fontFamily: "monospace",
    fontSize: "0.85rem",
    outline: "none",
  },
  btn: {
    padding: "0.6rem 1.2rem",
    background: "rgba(255,215,0,0.15)",
    border: "1px solid #FFD700",
    borderRadius: "4px",
    color: "#FFD700",
    fontFamily: "monospace",
    fontSize: "0.85rem",
    fontWeight: 700,
    cursor: "pointer",
    whiteSpace: "nowrap",
    transition: "all 0.2s",
  },
  activeBanner: {
    marginTop: "0.5rem",
    padding: "1rem",
    background: "rgba(255,68,68,0.08)",
    border: "1px solid rgba(255,68,68,0.3)",
    borderRadius: "6px",
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    flexWrap: "wrap",
    gap: "0.5rem",
    animation: "pulse-red 2s infinite",
  },
  activeInfo: {
    display: "flex",
    alignItems: "center",
    gap: "0.75rem",
    flexWrap: "wrap",
  },
  activeLabel: {
    fontSize: "0.8rem",
    color: "#FF4444",
    fontWeight: 700,
  },
  activeId: {
    fontSize: "0.85rem",
    color: "#fff",
    background: "rgba(0,0,0,0.3)",
    padding: "0.2rem 0.5rem",
    borderRadius: "3px",
  },
  targetService: {
    fontSize: "0.8rem",
    color: "#fff",
    background: "rgba(255,255,255,0.1)",
    padding: "0.2rem 0.5rem",
    borderRadius: "3px",
  },
  activeButtons: {
    display: "flex",
    gap: "0.5rem",
  },
  btnResolve: {
    padding: "0.6rem 1.2rem",
    background: "rgba(0,255,127,0.15)",
    border: "1px solid #00FF7F",
    borderRadius: "4px",
    color: "#00FF7F",
    fontFamily: "monospace",
    fontSize: "0.85rem",
    fontWeight: 700,
    cursor: "pointer",
  },
  btnAbandon: {
    padding: "0.6rem 1.2rem",
    background: "rgba(255,68,68,0.15)",
    border: "1px solid #FF4444",
    borderRadius: "4px",
    color: "#FF4444",
    fontFamily: "monospace",
    fontSize: "0.85rem",
    fontWeight: 700,
    cursor: "pointer",
  },
  tableWrap: {
    overflowX: "auto",
  },
  table: {
    width: "100%",
    borderCollapse: "collapse",
    fontSize: "0.8rem",
  },
  th: {
    textAlign: "left",
    padding: "0.5rem 0.75rem",
    borderBottom: "1px solid rgba(255,255,255,0.1)",
    color: "rgba(255,255,255,0.5)",
    textTransform: "uppercase",
    letterSpacing: "0.1rem",
    fontSize: "0.7rem",
    whiteSpace: "nowrap",
  },
  td: {
    padding: "0.4rem 0.75rem",
    borderBottom: "1px solid rgba(255,255,255,0.03)",
    color: "rgba(255,255,255,0.8)",
    whiteSpace: "nowrap",
  },
  statusBadge: {
    fontSize: "0.7rem",
    padding: "0.15rem 0.4rem",
    border: "1px solid",
    borderRadius: "3px",
    fontWeight: 700,
    letterSpacing: "0.05rem",
  },
  empty: {
    fontSize: "0.8rem",
    color: "rgba(255,255,255,0.3)",
    textAlign: "center",
    padding: "1rem",
  },
  scenarioGrid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
    gap: "0.75rem",
  },
  scenarioCard: {
    padding: "1rem",
    background: "rgba(0,0,0,0.3)",
    border: "1px solid rgba(255,255,255,0.06)",
    borderRadius: "6px",
  },
  scenarioHeader: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: "0.5rem",
    gap: "0.5rem",
  },
  scenarioName: {
    fontSize: "0.9rem",
    fontWeight: 700,
    color: "#fff",
  },
  scenarioMeta: {
    display: "flex",
    gap: "0.75rem",
    fontSize: "0.7rem",
    color: "rgba(255,255,255,0.5)",
    marginBottom: "0.5rem",
    flexWrap: "wrap",
  },
  scenarioDesc: {
    fontSize: "0.75rem",
    color: "rgba(255,255,255,0.6)",
    lineHeight: 1.4,
    margin: 0,
  },
  logContainer: {
    display: "flex",
    flexDirection: "column",
    gap: "0.2rem",
    maxHeight: "200px",
    overflowY: "auto",
    padding: "0.5rem",
    background: "rgba(0,0,0,0.3)",
    borderRadius: "4px",
  },
  logEmpty: {
    fontSize: "0.8rem",
    color: "rgba(255,255,255,0.2)",
    textAlign: "center",
    padding: "1rem",
  },
  logLine: {
    fontSize: "0.7rem",
    color: "rgba(255,255,255,0.5)",
    padding: "0.15rem 0.5rem",
    borderBottom: "1px solid rgba(255,255,255,0.02)",
    whiteSpace: "pre-wrap",
    wordBreak: "break-all",
  },
  badgeSeverity: (severity: string) => {
    const colorMap: Record<string, string> = {
      critical: "#FF4444",
      high: "#FF8C00",
      medium: "#FFD700",
      low: "#00FF7F",
    };
    return {
      fontSize: "0.65rem",
      padding: "0.15rem 0.4rem",
      background: `${colorMap[severity] ?? "#888"}22`,
      border: `1px solid ${colorMap[severity] ?? "#888"}`,
      borderRadius: "3px",
      color: colorMap[severity] ?? "#888",
      fontWeight: 700,
      letterSpacing: "0.05rem",
    };
  },
};

export default App;
