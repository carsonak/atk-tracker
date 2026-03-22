import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { HistoricalPoint } from "../lib/api";

interface Props {
  raw: HistoricalPoint[];
  summary: HistoricalPoint[];
}

export function HistoricalCharts({ raw, summary }: Props) {
  const rawData = raw.map((d) => ({
    timestamp: new Date(d.timestamp).toLocaleTimeString(),
    active_minutes: d.active_minutes ?? 0,
  }));

  const dailyData = summary.map((d) => ({
    day: new Date(d.timestamp).toLocaleDateString(),
    active_hours: d.active_hours ?? 0,
  }));

  return (
    <section className="card grid2">
      <div>
        <h2>Raw Activity: Minutes per Window</h2>
        <ResponsiveContainer width="100%" height={280}>
          <BarChart data={rawData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="timestamp" hide />
            <YAxis />
            <Tooltip />
            <Bar dataKey="active_minutes" fill="#ff6a3d" />
          </BarChart>
        </ResponsiveContainer>
      </div>
      <div>
        <h2>Daily Summaries: Hours per Day</h2>
        <ResponsiveContainer width="100%" height={280}>
          <BarChart data={dailyData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis dataKey="day" />
            <YAxis />
            <Tooltip />
            <Legend />
            <Bar dataKey="active_hours" fill="#1454d1" />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </section>
  );
}
