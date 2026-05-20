import { useState, useEffect } from "react";

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

interface IncidentAlarmProps {
  scenario: Scenario | null;
  remaining: number | null;
  elapsed: number | null;
  serviceName: string;
  serviceColor: string;
  onDismiss?: () => void;
}

const SEVERITY_COLORS: Record<string, string> = {
  critical: "#FF0044",
  high: "#FF4400",
  medium: "#FF8800",
  low: "#FFCC00",
};

function IncidentAlarm({
  scenario,
  remaining,
  elapsed,
  serviceName,
  serviceColor,
}: IncidentAlarmProps) {
  const [flash, setFlash] = useState(false);

  // Flashing effect
  useEffect(() => {
    const interval = setInterval(() => {
      setFlash((f) => !f);
    }, 800);
    return () => clearInterval(interval);
  }, []);

  if (!scenario) return null;

  const severityColor = SEVERITY_COLORS[scenario.severity] ?? "#FF4444";
  const progress = remaining != null && scenario.time_limit > 0
    ? ((scenario.time_limit - remaining) / scenario.time_limit) * 100
    : 0;
  const isUrgent = remaining != null && remaining <= 30;

  return (
    <div
      style={{
        ...styles.overlay,
        background: flash
          ? `radial-gradient(ellipse at center, ${severityColor}44 0%, ${severityColor}22 50%, transparent 80%)`
          : `radial-gradient(ellipse at center, ${severityColor}22 0%, transparent 60%, transparent 100%)`,
        borderColor: flash ? severityColor : `${severityColor}66`,
        boxShadow: flash
          ? `0 0 80px ${severityColor}44, inset 0 0 80px ${severityColor}22`
          : `0 0 40px ${severityColor}22`,
      }}
    >
      {/* Alarm header */}
      <div style={styles.alarmHeader}>
        <span style={{ ...styles.siren, opacity: flash ? 1 : 0.4 }}>
          🚨
        </span>
        <span style={styles.alarmTitle}>INCIDENTE ACTIVO</span>
        <span style={{ ...styles.siren, opacity: flash ? 1 : 0.4 }}>
          🚨
        </span>
      </div>

      {/* Target service */}
      <div style={styles.serviceRow}>
        <span
          style={{
            ...styles.serviceDot,
            background: serviceColor,
            boxShadow: flash
              ? `0 0 20px ${serviceColor}`
              : `0 0 10px ${serviceColor}66`,
          }}
        />
        <span style={styles.serviceName}>{serviceName}</span>
        <span
          style={{
            ...styles.failureBadge,
            background: `${severityColor}22`,
            borderColor: severityColor,
            color: severityColor,
          }}
        >
          {scenario.failure_type.toUpperCase()}
        </span>
        <span
          style={{
            ...styles.difficultyBadge,
            borderColor: severityColor,
            color: severityColor,
          }}
        >
          ⭐ {scenario.difficulty}/5
        </span>
      </div>

      {/* Scenario info */}
      <div style={styles.scenarioInfo}>
        <div style={styles.scenarioName}>{scenario.name}</div>
        <div style={styles.scenarioDesc}>{scenario.description}</div>
      </div>

      {/* Timer */}
      {remaining != null && (
        <div style={styles.timerSection}>
          <div style={styles.timerRow}>
            <span style={styles.timerLabel}>Tiempo restante</span>
            <span
              style={{
                ...styles.timerValue,
                color: isUrgent ? "#FF4444" : severityColor,
                animation: isUrgent && flash ? "none" : undefined,
              }}
            >
              {formatTime(remaining)}
            </span>
          </div>

          {/* Progress bar */}
          <div style={styles.progressTrack}>
            <div
              style={{
                ...styles.progressFill,
                width: `${Math.min(progress, 100)}%`,
                background: isUrgent
                  ? "#FF4444"
                  : `linear-gradient(90deg, ${severityColor}, ${serviceColor})`,
                boxShadow: isUrgent
                  ? "0 0 10px #FF4444"
                  : `0 0 10px ${severityColor}`,
              }}
            />
          </div>

          {elapsed != null && (
            <div style={styles.elapsed}>
              Tiempo transcurrido: {formatTime(elapsed)}
            </div>
          )}

          {/* Hints */}
          {scenario.hints.length > 0 && (
            <div style={styles.hints}>
              <div style={styles.hintsLabel}>💡 Pistas</div>
              {scenario.hints.map((h, i) => (
                <div key={i} style={styles.hintItem}>
                  {i + 1}. {h}
                </div>
              ))}
            </div>
          )}

          {isUrgent && (
            <div style={styles.urgentWarning}>
              ⚠️ ¡TIEMPO CRÍTICO! ⚠️
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}

const styles: Record<string, React.CSSProperties | any> = {
  overlay: {
    position: "fixed",
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    zIndex: 1000,
    border: "2px solid",
    borderRadius: 0,
    transition: "all 0.3s ease",
    padding: "2rem",
  },
  alarmHeader: {
    display: "flex",
    alignItems: "center",
    gap: "1rem",
    marginBottom: "1.5rem",
  },
  siren: {
    fontSize: "2rem",
    transition: "opacity 0.3s ease",
  },
  alarmTitle: {
    fontSize: "2.5rem",
    fontWeight: 900,
    color: "#FF4444",
    letterSpacing: "0.3rem",
    textShadow: "0 0 20px rgba(255,68,68,0.5)",
  },
  serviceRow: {
    display: "flex",
    alignItems: "center",
    gap: "1rem",
    marginBottom: "1rem",
    flexWrap: "wrap",
    justifyContent: "center",
  },
  serviceDot: {
    width: 16,
    height: 16,
    borderRadius: "50%",
    transition: "all 0.3s ease",
  },
  serviceName: {
    fontSize: "1.8rem",
    fontWeight: 700,
    color: "#fff",
    textTransform: "uppercase",
    letterSpacing: "0.2rem",
  },
  failureBadge: {
    fontSize: "0.8rem",
    padding: "0.3rem 0.6rem",
    border: "1px solid",
    borderRadius: "4px",
    fontWeight: 700,
    letterSpacing: "0.1rem",
  },
  difficultyBadge: {
    fontSize: "0.8rem",
    padding: "0.3rem 0.6rem",
    border: "1px solid",
    borderRadius: "4px",
  },
  scenarioInfo: {
    textAlign: "center",
    marginBottom: "1.5rem",
  },
  scenarioName: {
    fontSize: "1.2rem",
    color: "rgba(255,255,255,0.8)",
    fontWeight: 700,
    marginBottom: "0.5rem",
  },
  scenarioDesc: {
    fontSize: "0.9rem",
    color: "rgba(255,255,255,0.6)",
    maxWidth: "600px",
    lineHeight: 1.5,
  },
  timerSection: {
    width: "100%",
    maxWidth: "500px",
  },
  timerRow: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: "0.5rem",
  },
  timerLabel: {
    fontSize: "0.9rem",
    color: "rgba(255,255,255,0.6)",
    textTransform: "uppercase",
    letterSpacing: "0.1rem",
  },
  timerValue: {
    fontSize: "3rem",
    fontWeight: 900,
    fontFamily: "'Courier New', Courier, monospace",
    letterSpacing: "0.2rem",
  },
  progressTrack: {
    width: "100%",
    height: "8px",
    background: "rgba(255,255,255,0.1)",
    borderRadius: "4px",
    overflow: "hidden",
    marginBottom: "0.5rem",
  },
  progressFill: {
    height: "100%",
    borderRadius: "4px",
    transition: "width 1s linear",
  },
  elapsed: {
    fontSize: "0.75rem",
    color: "rgba(255,255,255,0.4)",
    textAlign: "center",
    marginBottom: "0.75rem",
  },
  hints: {
    marginTop: "1rem",
    textAlign: "left",
    background: "rgba(0,0,0,0.3)",
    borderRadius: "6px",
    padding: "0.75rem 1rem",
  },
  hintsLabel: {
    fontSize: "0.8rem",
    color: "#FFD700",
    marginBottom: "0.5rem",
    fontWeight: 700,
  },
  hintItem: {
    fontSize: "0.8rem",
    color: "rgba(255,255,255,0.7)",
    marginBottom: "0.25rem",
    lineHeight: 1.4,
  },
  urgentWarning: {
    marginTop: "1rem",
    fontSize: "1.2rem",
    fontWeight: 900,
    color: "#FF4444",
    textAlign: "center",
    animation: "pulse-red 0.5s infinite",
    letterSpacing: "0.2rem",
  },
};

export default IncidentAlarm;
