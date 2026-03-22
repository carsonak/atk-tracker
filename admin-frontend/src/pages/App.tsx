import { useEffect, useMemo, useState } from "react";
import { HistoricalCharts } from "../components/HistoricalCharts";
import { LiveTable } from "../components/LiveTable";
import { fetchLive, fetchStats, HistoricalPoint, LivePresence } from "../lib/api";

export function App() {
  const [liveRows, setLiveRows] = useState<LivePresence[]>([]);
  const [raw, setRaw] = useState<HistoricalPoint[]>([]);
  const [summary, setSummary] = useState<HistoricalPoint[]>([]);
  const [apprenticeID, setApprenticeID] = useState("uid-1000");

  const dates = useMemo(() => {
    const now = new Date();
    const from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
    return {
      from: from.toISOString().slice(0, 10),
      to: now.toISOString().slice(0, 10),
    };
  }, []);

  useEffect(() => {
    const poll = async () => {
      const data = await fetchLive();
      setLiveRows(data);
    };
    poll();
    const id = setInterval(poll, 10000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    const load = async () => {
      const data = await fetchStats(apprenticeID, dates.from, dates.to);
      setRaw(data.raw);
      setSummary(data.summary);
    };
    load();
  }, [apprenticeID, dates.from, dates.to]);

  return (
    <main>
      <header className="hero">
        <p className="eyebrow">Zone01 Cluster</p>
        <h1>ATK Operations Dashboard</h1>
        <div className="toolbar">
          <label>
            Apprentice ID
            <input value={apprenticeID} onChange={(e) => setApprenticeID(e.target.value)} />
          </label>
        </div>
      </header>
      <LiveTable rows={liveRows} />
      <HistoricalCharts raw={raw} summary={summary} />
    </main>
  );
}
