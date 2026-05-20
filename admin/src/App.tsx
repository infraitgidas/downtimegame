import { useState } from "react";
import useWebSocket from "./useWebSocket";

function App() {
  const [wsStatus, setWsStatus] = useState<"connecting" | "connected" | "disconnected">("disconnected");
  const [logs, setLogs] = useState<string[]>([]);

  const handleMessage = (data: unknown) => {
    const msg = typeof data === "string" ? data : JSON.stringify(data);
    setLogs((prev) => [msg, ...prev].slice(0, 50));
  };

  const { isConnected } = useWebSocket("ws://localhost:8080/ws", handleMessage);

  useState(() => {
    setWsStatus(isConnected ? "connected" : "disconnected");
  });

  return (
    <div style={styles.container}>
      <header style={styles.header}>
        <h1 style={styles.title}>Downtime Game — Admin</h1>
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
      </header>

      <main style={styles.main}>
        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>Controles de Juego</h2>
          <p style={styles.placeholder}>
            Panel de control para iniciar/detener partidas, seleccionar escenarios
            de downtime, y monitorear jugadores.
          </p>
          <div style={styles.placeholderGrid}>
            <div style={styles.placeholderCard}>▶ Iniciar Partida</div>
            <div style={styles.placeholderCard}>⏹ Detener Partida</div>
            <div style={styles.placeholderCard}>🎲 Escenario Aleatorio</div>
            <div style={styles.placeholderCard}>📊 Estadísticas</div>
          </div>
        </section>

        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>Logs de WebSocket</h2>
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
      </main>
    </div>
  );
}

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
  },
  title: {
    fontSize: "1.5rem",
    fontWeight: 700,
    color: "#FFD700",
    textTransform: "uppercase",
    letterSpacing: "0.2rem",
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
  main: {
    display: "flex",
    flexDirection: "column",
    gap: "2rem",
    maxWidth: "1200px",
    margin: "0 auto",
  },
  section: {
    background: "rgba(255,255,255,0.02)",
    border: "1px solid rgba(255,255,255,0.08)",
    borderRadius: "8px",
    padding: "1.5rem",
  },
  sectionTitle: {
    fontSize: "1rem",
    color: "#FFD700",
    marginBottom: "1rem",
    textTransform: "uppercase",
    letterSpacing: "0.15rem",
  },
  placeholder: {
    fontSize: "0.85rem",
    color: "rgba(255,255,255,0.5)",
    marginBottom: "1rem",
    lineHeight: 1.5,
  },
  placeholderGrid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
    gap: "1rem",
  },
  placeholderCard: {
    padding: "1.5rem",
    background: "rgba(255,255,255,0.03)",
    border: "1px solid rgba(255,255,255,0.06)",
    borderRadius: "6px",
    textAlign: "center",
    fontSize: "0.9rem",
    color: "rgba(255,255,255,0.6)",
    cursor: "default",
    transition: "all 0.2s",
  },
  logContainer: {
    display: "flex",
    flexDirection: "column",
    gap: "0.25rem",
    maxHeight: "300px",
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
    fontSize: "0.75rem",
    color: "rgba(255,255,255,0.6)",
    padding: "0.2rem 0.5rem",
    borderBottom: "1px solid rgba(255,255,255,0.03)",
  },
};

export default App;
