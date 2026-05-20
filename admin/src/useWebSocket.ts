import { useEffect, useRef, useState, useCallback } from "react";

type MessageHandler = (data: unknown) => void;

interface UseWebSocketReturn {
  isConnected: boolean;
  sendMessage: (msg: string) => void;
}

function useWebSocket(url: string, onMessage: MessageHandler): UseWebSocketReturn {
  const wsRef = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;

  const sendMessage = useCallback((msg: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(msg);
    }
  }, []);

  useEffect(() => {
    function connect() {
      const ws = new WebSocket(url);

      ws.onopen = () => setIsConnected(true);
      ws.onclose = () => {
        setIsConnected(false);
        setTimeout(connect, 3000);
      };
      ws.onerror = () => ws.close();
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          onMessageRef.current(data);
        } catch {
          onMessageRef.current(event.data);
        }
      };

      wsRef.current = ws;
    }

    connect();
    return () => wsRef.current?.close();
  }, [url]);

  return { isConnected, sendMessage };
}

export default useWebSocket;
