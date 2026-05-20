import { useState, useEffect } from "react";
import ServiceCard from "./components/ServiceCard";
import useWebSocket from "./hooks/useWebSocket";

interface Service {
  id: string;
  name: string;
  color: string;
  ip: string;
  port: number;
}

const DEFAULT_SERVICES: Service[] = [
  { id: "sg-rojo", name: "Rojo", color: "#DC143C", ip: "192.168.1.200", port: 8080 },
  { id: "sg-azul", name: "Azul", color: "#1E90FF", ip: "192.168.1.204", port: 8084 },
  { id: "sg-verde", name: "Verde", color: "#00FF7F", ip: "192.168.1.202", port: 8082 },
  { id: "sg-amarillo", name: "Amarillo", color: "#FFD700", ip: "192.168.1.203", port: 8083 },
];

function App() {
  const [services] = useState<Service[]>(DEFAULT_SERVICES);
  const [wsStatus, setWsStatus] = useState<"connecting" | "connected" | "disconnected">("disconnected");
  const [lastMessage, setLastMessage] = useState<string | null>(null);

  const handleMessage = (data: unknown) => {
    setLastMessage(JSON.stringify(data));
  };

  const { isConnected } = useWebSocket("ws://localhost:8080/ws", handleMessage);

  useEffect(() => {
    setWsStatus(isConnected ? "connected" : "disconnected");
  }, [isConnected]);

  return (
    <div style={styles.container}>
      <header style={styles.header}>
        <h1 style={styles.title}>Downtime Game — Dashboard</h1>
        <div style={styles.statusBar}>
          <span style={styles.statusLabel}>WebSocket:</span>
          <span
            style={{
              ...styles.statusDot,
              background: wsStatus === "connected" ? "#00FF7F" : "#FF4444",
            }}
          />
          <span style={styles.statusText}>{wsStatus}</span>
        </div>
      </header>

      <main style={styles.grid}>
        {services.map((svc) => (
          <ServiceCard key={svc.id} service={svc} />
        ))}
      </main>

      {lastMessage && (
        <footer style={styles.footer}>
          <code>Last WS: {lastMessage}</code>
        </footer>
      )}
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
    color: "#00FF7F",
    textTransform: "uppercase",
    letterSpacing: "0.2rem",
  },
  statusBar: {
    display: "flex",
    alignItems: "center",
    gap: "0.5rem",
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
