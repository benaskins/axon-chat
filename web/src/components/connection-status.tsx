interface ConnectionStatusProps {
  status: "connected" | "reconnecting" | "disconnected";
}

export function ConnectionStatus({ status }: ConnectionStatusProps) {
  if (status === "connected") return null;

  return (
    <div className="fixed top-0 left-0 right-0 z-50 bg-destructive px-4 py-2 text-center text-sm text-white">
      {status === "reconnecting" ? "Reconnecting..." : "Connection lost"}
    </div>
  );
}
