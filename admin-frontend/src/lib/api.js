const baseUrl = import.meta.env.VITE_API_URL ?? "http://localhost:8080";
export async function fetchLive() {
    const res = await fetch(`${baseUrl}/live`);
    if (!res.ok) {
        throw new Error("Failed to fetch live view");
    }
    return res.json();
}
export async function fetchStats(apprenticeId, from, to) {
    const query = new URLSearchParams({ apprentice_id: apprenticeId, from, to });
    const res = await fetch(`${baseUrl}/stats?${query.toString()}`);
    if (!res.ok) {
        throw new Error("Failed to fetch historical stats");
    }
    return res.json();
}
