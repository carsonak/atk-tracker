import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Bar, BarChart, CartesianGrid, Legend, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
export function HistoricalCharts({ raw, summary }) {
    const rawData = raw.map((d) => ({
        timestamp: new Date(d.timestamp).toLocaleTimeString(),
        active_minutes: d.active_minutes ?? 0,
    }));
    const dailyData = summary.map((d) => ({
        day: new Date(d.timestamp).toLocaleDateString(),
        active_hours: d.active_hours ?? 0,
    }));
    return (_jsxs("section", { className: "card grid2", children: [_jsxs("div", { children: [_jsx("h2", { children: "Raw Activity: Minutes per Window" }), _jsx(ResponsiveContainer, { width: "100%", height: 280, children: _jsxs(BarChart, { data: rawData, children: [_jsx(CartesianGrid, { strokeDasharray: "3 3" }), _jsx(XAxis, { dataKey: "timestamp", hide: true }), _jsx(YAxis, {}), _jsx(Tooltip, {}), _jsx(Bar, { dataKey: "active_minutes", fill: "#ff6a3d" })] }) })] }), _jsxs("div", { children: [_jsx("h2", { children: "Daily Summaries: Hours per Day" }), _jsx(ResponsiveContainer, { width: "100%", height: 280, children: _jsxs(BarChart, { data: dailyData, children: [_jsx(CartesianGrid, { strokeDasharray: "3 3" }), _jsx(XAxis, { dataKey: "day" }), _jsx(YAxis, {}), _jsx(Tooltip, {}), _jsx(Legend, {}), _jsx(Bar, { dataKey: "active_hours", fill: "#1454d1" })] }) })] })] }));
}
