import { useState, useEffect, useCallback, useRef } from "react";
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

interface DemoStatus {
  running: boolean;
  round: number;
  game_id?: string;
  scenario?: string;
  started_at?: string;
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

async function apiDelete<T>(path: string): Promise<T> {
  const res = await fetch(path, { method: "DELETE" });
  if (!res.ok) {
    const errBody = await res.text();
    throw new Error(`DELETE ${path}: ${res.status} — ${errBody}`);
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
  const [selectedScenarioId, setSelectedScenarioId] = useState("");
  const [activeGameId, setActiveGameId] = useState<string | null>(null);
  const [activeScenario, setActiveScenario] = useState<Scenario | null>(null);
  const [actionMsg, setActionMsg] = useState<string | null>(null);

  // Confirmation state
  const [confirmReset, setConfirmReset] = useState(false);

  // Demo mode
  const [demoRunning, setDemoRunning] = useState(false);
  const [demoRound, setDemoRound] = useState(0);

  // Logs
  const [logs, setLogs] = useState<string[]>([]);

  // ── Fetch helpers ─────────────────────────────────────────────────────────

  const refreshAll = useCallback(() => {
    apiGet<Game[]>("/api/games").then(setGames).catch(console.error);
    apiGet<LeaderboardEntry[]>("/api/leaderboard").then(setLeaderboard).catch(console.error);
  }, []);

  const showMsg = useCallback((msg: string) => {
    setActionMsg(msg);
    setTimeout(() => setActionMsg(null), 5000);
  }, []);

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
        refreshAll();
        showMsg(`${now} — Partida creada`);
        break;

      case "game_started": {
        setActiveGameId((evt.game as Game)?.id ?? null);
        setActiveScenario(evt.scenario as Scenario ?? null);
        showMsg(`${now} — Partida INICIADA`);
        refreshAll();
        break;
      }

      case "game_completed":
        setActiveGameId(null);
        setActiveScenario(null);
        showMsg(`${now} — Partida COMPLETADA (score: ${evt.score})`);
        refreshAll();
        break;

      case "game_abandoned":
      case "game_timeout":
        setActiveGameId(null);
        setActiveScenario(null);
        showMsg(`${now} — Partida ${evt.type === "game_abandoned" ? "ABANDONADA" : "TIMEOUT"}`);
        refreshAll();
        break;

      case "incident_resolved":
        showMsg(`${now} — Incidente resuelto (score: ${evt.score})`);
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
      showMsg(`Error: ${err}`);
    }
  }, [playerName, showMsg]);

  const handleStartGame = useCallback(async () => {
    if (!selectedGameId) return;
    try {
      const body: Record<string, string> = {};
      if (selectedScenarioId) {
        body.scenario_id = selectedScenarioId;
      }
      await apiPost(`/api/games/${selectedGameId}/start`, body);
      setSelectedGameId("");
      setSelectedScenarioId("");
    } catch (err) {
      showMsg(`Error: ${err}`);
    }
  }, [selectedGameId, selectedScenarioId, showMsg]);

  const handleResolveIncident = useCallback(async () => {
    if (!activeGameId) return;
    try {
      await apiPost(`/api/games/${activeGameId}/resolve`);
    } catch (err) {
      showMsg(`Error: ${err}`);
    }
  }, [activeGameId, showMsg]);

  const handleAbandonGame = useCallback(async (gameId?: string) => {
    const id = gameId ?? activeGameId;
    if (!id) return;
    try {
      await apiPost(`/api/games/${id}/abandon`);
      if (!gameId) showMsg("Partida abandonada");
    } catch (err) {
      showMsg(`Error: ${err}`);
    }
  }, [activeGameId, showMsg]);

  const handleDeleteGame = useCallback(async (gameId: string) => {
    try {
      await apiDelete(`/api/games/${gameId}`);
      showMsg("Partida eliminada");
      refreshAll();
    } catch (err) {
      showMsg(`Error: ${err}`);
    }
  }, [showMsg, refreshAll]);

  const handleRestartGame = useCallback(async (game: Game) => {
    // Create a new game with the same player name
    try {
      await apiPost("/api/games", { player_name: game.player_name });
      showMsg(`Nueva partida creada para ${game.player_name}`);
      refreshAll();
    } catch (err) {
      showMsg(`Error: ${err}`);
    }
  }, [showMsg, refreshAll]);

  const handleAdminReset = useCallback(async () => {
    try {
      await apiPost("/api/admin/reset");
      setActiveGameId(null);
      setActiveScenario(null);
      setConfirmReset(false);
      showMsg("🔄 Reset global completado");
      refreshAll();
    } catch (err) {
      showMsg(`Error: ${err}`);
    }
  }, [showMsg, refreshAll]);

  // ── Demo Mode ────────────────────────────────────────────────────────────
  const pollDemoRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const handleDemoStart = useCallback(async () => {
    try {
      await apiPost("/api/admin/demo");
      setDemoRunning(true);
      showMsg("🎮 Modo demo iniciado");
      // Poll demo status
      if (pollDemoRef.current) clearInterval(pollDemoRef.current);
      pollDemoRef.current = setInterval(async () => {
        try {
          const status = await apiGet<DemoStatus>("/api/admin/demo");
          setDemoRunning(status.running);
          setDemoRound(status.round);
          if (status.scenario) {
            setActiveScenario({
              id: status.game_id ?? "",
              name: status.scenario,
              description: "",
              target_service_id: "",
              failure_type: "",
              severity: "high",
              difficulty: 0,
              hints: [],
              fix_hint: "",
              time_limit: 0,
            });
          }
        } catch { /* ignore */ }
      }, 2000);
    } catch (err) {
      showMsg(`Error: ${err}`);
    }
  }, [showMsg]);

  const handleDemoStop = useCallback(async () => {
    try {
      await apiPost("/api/admin/demo/stop");
      setDemoRunning(false);
      setDemoRound(0);
      setActiveScenario(null);
      showMsg("⏹ Modo demo detenido");
      if (pollDemoRef.current) {
        clearInterval(pollDemoRef.current);
        pollDemoRef.current = null;
      }
      refreshAll();
    } catch (err) {
      showMsg(`Error: ${err}`);
    }
  }, [showMsg, refreshAll]);

  // Cleanup poll on unmount
  useEffect(() => {
    return () => {
      if (pollDemoRef.current) clearInterval(pollDemoRef.current);
    };
  }, []);

  const formatDate = (s: string | null) => {
    if (!s) return "—";
    return new Date(s).toLocaleString();
  };

  const activeGames = games.filter((g) => g.status === "active");

  return (
    <div style={styles.container}>
      <header style={styles.header}>
        <div style={styles.headerLeft}>
          <img
            src="/assets/logo-gidas.png"
            alt="GIDAS"
            style={styles.logoGidas}
          />
          <h1 style={styles.title}>Downtime Game — Admin</h1>
        </div>
        <div style={styles.headerRight}>
          <img
            src="/assets/logo_infra_blanco.png"
            alt="INFRA IT"
            style={styles.logoInfra}
          />
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

          {/* Start Game — with scenario selector */}
          <div style={styles.controlRow}>
            <select
              style={styles.select}
              value={selectedGameId}
              onChange={(e) => setSelectedGameId(e.target.value)}
            >
              <option value="">-- Partida pendiente --</option>
              {pendingGames.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.id.slice(0, 8)} — {g.player_name}
                </option>
              ))}
            </select>
            <select
              style={styles.selectSmall}
              value={selectedScenarioId}
              onChange={(e) => setSelectedScenarioId(e.target.value)}
              disabled={!selectedGameId}
            >
              <option value="">🎲 Aleatorio</option>
              {scenarios.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} ({SERVICE_NAMES[s.target_service_id] ?? s.target_service_id})
                </option>
              ))}
            </select>
            <button
              style={styles.btn}
              onClick={handleStartGame}
              disabled={!selectedGameId}
            >
              ▶ Iniciar
            </button>
          </div>

          {/* Active game banner */}
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
                  ✅ Resolver
                </button>
                <button style={styles.btnAbandon} onClick={() => handleAbandonGame()}>
                  ⏹ Abandonar
                </button>
              </div>
            </div>
          )}

          {/* Active games count + Reset */}
          <div style={styles.controlRow}>
            <div style={styles.infoRow}>
              <span style={styles.infoItem}>
                🎮 Partidas: <strong>{games.length}</strong>
              </span>
              <span style={styles.infoItem}>
                🔴 Activas: <strong style={{ color: "#FF4444" }}>{activeGames.length}</strong>
              </span>
              <span style={styles.infoItem}>
                ⏳ Pendientes: <strong style={{ color: "#FFD700" }}>{pendingGames.length}</strong>
              </span>
            </div>
            <div style={styles.flex1} />
            {demoRunning ? (
              <button
                style={styles.btnDemoActive}
                onClick={handleDemoStop}
              >
                ⏹ Demo R{demoRound}
              </button>
            ) : (
              <button
                style={styles.btnDemo}
                onClick={handleDemoStart}
              >
                🎮 Demo
              </button>
            )}
            {!confirmReset ? (
              <button
                style={styles.btnDanger}
                onClick={() => setConfirmReset(true)}
              >
                🛑 Reset
              </button>
            ) : (
              <div style={styles.confirmRow}>
                <span style={styles.confirmText}>¿Resetear todo?</span>
                <button style={styles.btnDanger} onClick={handleAdminReset}>
                  ✓ Conf
                </button>
                <button
                  style={styles.btnCancel}
                  onClick={() => setConfirmReset(false)}
                >
                  ✕
                </button>
              </div>
            )}
          </div>
        </section>

        {/* ── Games List ── */}
        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>Partidas ({games.length})</h2>
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
                  <th style={styles.th}>Acciones</th>
                </tr>
              </thead>
              <tbody>
                {games.length === 0 ? (
                  <tr>
                    <td style={styles.td} colSpan={7}>
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
                      <td style={styles.td}>
                        <div style={styles.actionCell}>
                          {g.status === "active" ? (
                            <button
                              style={styles.btnSmallDanger}
                              onClick={() => handleAbandonGame(g.id)}
                              title="Detener partida"
                            >
                              ⏹
                            </button>
                          ) : (
                            <>
                              <button
                                style={styles.btnSmallRestart}
                                onClick={() => handleRestartGame(g)}
                                title="Reiniciar (mismo jugador)"
                              >
                                🔄
                              </button>
                              <button
                                style={styles.btnSmallDelete}
                                onClick={() => handleDeleteGame(g.id)}
                                title="Eliminar partida"
                              >
                                🗑
                              </button>
                            </>
                          )}
                        </div>
                      </td>
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

        {/* ── Footer ── */}
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
            <span style={styles.footerVersion}>
              v1.0 · {games.length} partidas · {leaderboard.length} scores
            </span>
          </div>
        </footer>
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
    gap: "0.75rem",
  },
  headerLeft: {
    display: "flex",
    alignItems: "center",
    gap: "1rem",
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
    alignItems: "center",
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
  selectSmall: {
    flex: 1,
    minWidth: "200px",
    maxWidth: "300px",
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
  btnDemo: {
    padding: "0.6rem 1.2rem",
    background: "rgba(0,255,127,0.12)",
    border: "1px solid #00FF7F",
    borderRadius: "4px",
    color: "#00FF7F",
    fontFamily: "monospace",
    fontSize: "0.85rem",
    fontWeight: 700,
    cursor: "pointer",
    whiteSpace: "nowrap",
    animation: "pulse-green 2s infinite",
  },
  btnDemoActive: {
    padding: "0.6rem 1.2rem",
    background: "rgba(255,68,68,0.15)",
    border: "1px solid #FF4444",
    borderRadius: "4px",
    color: "#FF4444",
    fontFamily: "monospace",
    fontSize: "0.85rem",
    fontWeight: 700,
    cursor: "pointer",
    whiteSpace: "nowrap",
    animation: "pulse-red 1s infinite",
  },
  btnDanger: {
    padding: "0.6rem 1.2rem",
    background: "rgba(255,68,68,0.15)",
    border: "1px solid #FF4444",
    borderRadius: "4px",
    color: "#FF4444",
    fontFamily: "monospace",
    fontSize: "0.85rem",
    fontWeight: 700,
    cursor: "pointer",
    whiteSpace: "nowrap",
    transition: "all 0.2s",
  },
  btnCancel: {
    padding: "0.4rem 0.8rem",
    background: "rgba(255,255,255,0.05)",
    border: "1px solid rgba(255,255,255,0.2)",
    borderRadius: "4px",
    color: "rgba(255,255,255,0.6)",
    fontFamily: "monospace",
    fontSize: "0.8rem",
    cursor: "pointer",
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
  infoRow: {
    display: "flex",
    gap: "1rem",
    flexWrap: "wrap",
  },
  infoItem: {
    fontSize: "0.8rem",
    color: "rgba(255,255,255,0.5)",
  },
  flex1: {
    flex: 1,
  },
  confirmRow: {
    display: "flex",
    alignItems: "center",
    gap: "0.5rem",
  },
  confirmText: {
    fontSize: "0.8rem",
    color: "#FF4444",
    fontWeight: 700,
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
  actionCell: {
    display: "flex",
    gap: "0.3rem",
    alignItems: "center",
  },
  btnSmallDanger: {
    padding: "0.2rem 0.4rem",
    background: "rgba(255,68,68,0.15)",
    border: "1px solid rgba(255,68,68,0.4)",
    borderRadius: "3px",
    color: "#FF4444",
    cursor: "pointer",
    fontSize: "0.8rem",
    lineHeight: 1,
  },
  btnSmallRestart: {
    padding: "0.2rem 0.4rem",
    background: "rgba(0,255,127,0.1)",
    border: "1px solid rgba(0,255,127,0.3)",
    borderRadius: "3px",
    color: "#00FF7F",
    cursor: "pointer",
    fontSize: "0.8rem",
    lineHeight: 1,
  },
  btnSmallDelete: {
    padding: "0.2rem 0.4rem",
    background: "rgba(255,68,68,0.1)",
    border: "1px solid rgba(255,68,68,0.3)",
    borderRadius: "3px",
    color: "#FF4444",
    cursor: "pointer",
    fontSize: "0.8rem",
    lineHeight: 1,
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

  // ── Footer ──
  footer: {
    marginTop: "1rem",
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
  footerVersion: {
    fontSize: "0.6rem",
    color: "rgba(255,255,255,0.15)",
    letterSpacing: "0.05rem",
  },
};

export default App;
