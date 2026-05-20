import { useState, useEffect } from "react";

interface Service {
  id: string;
  name: string;
  color: string;
  ip: string;
  port: number;
}

interface ServiceCardProps {
  service: Service;
  hasIncident?: boolean;
  isAffected?: boolean;
}

interface HealthData {
  status: string;
  service: string;
  color: string;
  uptime: string;
  uptime_seconds: number;
}

function ServiceCard({ service, hasIncident = false, isAffected = false }: ServiceCardProps) {
  const [health, setHealth] = useState<HealthData | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const checkHealth = async () => {
      try {
        const res = await fetch(`http://${service.ip}:${service.port}/health`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const data: HealthData = await res.json();
        setHealth(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unknown error");
        setHealth(null);
      }
    };

    checkHealth();
    const interval = setInterval(checkHealth, 5000);
    return () => clearInterval(interval);
  }, [service.ip, service.port]);

  const isOnline = health?.status === "ok";

  // Determine border color and status
  let borderColor = service.color;
  let statusText = isOnline ? "ONLINE" : "OFFLINE";
  let statusColor = isOnline ? service.color : "#FF4444";
  let statusDetail = isOnline ? "Saludable" : "Caído";

  if (hasIncident && isAffected) {
    borderColor = "#FF0044";
    statusText = "🚨 INCIDENTE";
    statusColor = "#FF0044";
    statusDetail = "Servicio afectado";
  } else if (hasIncident) {
    // Incident on another service — this one might be fine
  }

  return (
    <div
      style={{
        ...styles.card,
        borderColor,
        boxShadow: hasIncident && isAffected
          ? `0 0 40px rgba(255,0,68,0.3), inset 0 0 40px rgba(255,0,68,0.1)`
          : isOnline
          ? `0 0 20px ${service.color}22`
          : "0 0 20px rgba(255,68,68,0.2)",
      }}
    >
      <div style={styles.header}>
        <span
          style={{
            ...styles.dot,
            background: hasIncident && isAffected ? "#FF0044" : isOnline ? service.color : "#FF4444",
            boxShadow: hasIncident && isAffected
              ? "0 0 12px #FF0044, 0 0 24px #FF004488"
              : `0 0 8px currentColor`,
          }}
        />
        <span style={styles.name}>{service.name}</span>
        <span style={{ ...styles.status, color: statusColor }}>
          {statusText}
        </span>
      </div>

      <div style={styles.id}>{service.id}</div>

      <div style={styles.grid}>
        <div style={styles.item}>
          <div style={styles.label}>IP</div>
          <div style={styles.value}>{service.ip}</div>
        </div>
        <div style={styles.item}>
          <div style={styles.label}>Puerto</div>
          <div style={styles.value}>{service.port}</div>
        </div>
        {health && (
          <>
            <div style={styles.item}>
              <div style={styles.label}>Uptime</div>
              <div style={{ ...styles.value, color: service.color }}>{health.uptime}</div>
            </div>
            <div style={styles.item}>
              <div style={styles.label}>Estado</div>
              <div style={{ ...styles.value, color: statusColor }}>
                {statusDetail}
              </div>
            </div>
          </>
        )}
      </div>

      {error && (
        <div style={styles.error}>
          <div style={styles.errorLabel}>Error de conexión</div>
          {error}
        </div>
      )}

      {hasIncident && isAffected && (
        <div style={styles.incidentBanner}>
          ⚠️ Incidente activo — este servicio está siendo atacado
        </div>
      )}

      {/* Branding watermark */}
      <div style={styles.brandWatermark}>
        <img
          src="/assets/logo-gidas.png"
          alt="GIDAS"
          style={styles.brandLogo}
        />
      </div>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  card: {
    background: "linear-gradient(135deg, #12121a 0%, #0a0a0f 100%)",
    border: "1px solid",
    borderRadius: "8px",
    padding: "1.5rem",
    transition: "all 0.3s ease",
    position: "relative",
    overflow: "hidden",
  },
  header: {
    display: "flex",
    alignItems: "center",
    gap: "0.75rem",
    marginBottom: "0.5rem",
  },
  dot: {
    width: 12,
    height: 12,
    borderRadius: "50%",
    transition: "all 0.3s ease",
  },
  name: {
    fontSize: "1.5rem",
    fontWeight: 700,
    textTransform: "uppercase",
    letterSpacing: "0.2rem",
  },
  status: {
    marginLeft: "auto",
    fontSize: "0.8rem",
    fontWeight: 700,
    letterSpacing: "0.15rem",
  },
  id: {
    fontSize: "0.75rem",
    color: "rgba(255,255,255,0.3)",
    marginBottom: "1rem",
  },
  grid: {
    display: "grid",
    gridTemplateColumns: "1fr 1fr",
    gap: "1px",
    background: "rgba(255,255,255,0.06)",
    borderRadius: "4px",
    overflow: "hidden",
  },
  item: {
    background: "rgba(0,0,0,0.4)",
    padding: "0.6rem 1rem",
  },
  label: {
    fontSize: "0.6rem",
    color: "rgba(255,255,255,0.35)",
    textTransform: "uppercase",
    letterSpacing: "0.1rem",
    marginBottom: "0.2rem",
  },
  value: {
    fontSize: "0.9rem",
    color: "#ffffff",
    fontFamily: "monospace",
  },
  error: {
    marginTop: "0.75rem",
    padding: "0.5rem",
    background: "rgba(255,68,68,0.1)",
    border: "1px solid rgba(255,68,68,0.3)",
    borderRadius: "4px",
    fontSize: "0.75rem",
    color: "#FF4444",
    textAlign: "center",
  },
  errorLabel: {
    fontSize: "0.65rem",
    color: "rgba(255,68,68,0.6)",
    textTransform: "uppercase",
    letterSpacing: "0.1rem",
    marginBottom: "0.2rem",
  },
  incidentBanner: {
    marginTop: "0.75rem",
    padding: "0.5rem",
    background: "rgba(255,0,68,0.1)",
    border: "1px solid rgba(255,0,68,0.3)",
    borderRadius: "4px",
    fontSize: "0.7rem",
    color: "#FF0044",
    textAlign: "center",
    fontWeight: 700,
    letterSpacing: "0.05rem",
  },
  brandWatermark: {
    position: "absolute",
    bottom: "6px",
    right: "8px",
    opacity: 0.12,
    pointerEvents: "none",
  },
  brandLogo: {
    height: "16px",
    width: "auto",
  },
};

export default ServiceCard;
