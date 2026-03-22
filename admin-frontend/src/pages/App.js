import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect, useMemo, useState } from "react";
import { HistoricalCharts } from "../components/HistoricalCharts";
import { LiveTable } from "../components/LiveTable";
import { fetchLive, fetchStats } from "../lib/api";
export function App() {
    const [liveRows, setLiveRows] = useState([]);
    const [raw, setRaw] = useState([]);
    const [summary, setSummary] = useState([]);
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
    return (_jsxs("main", { children: [_jsxs("header", { className: "hero", children: [_jsx("p", { className: "eyebrow", children: "Zone01 Cluster" }), _jsx("h1", { children: "ATK Operations Dashboard" }), _jsx("div", { className: "toolbar", children: _jsxs("label", { children: ["Apprentice ID", _jsx("input", { value: apprenticeID, onChange: (e) => setApprenticeID(e.target.value) })] }) })] }), _jsx(LiveTable, { rows: liveRows }), _jsx(HistoricalCharts, { raw: raw, summary: summary })] }));
}
