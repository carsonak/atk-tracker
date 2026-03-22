import { LivePresence } from "../lib/api";

interface Props {
  rows: LivePresence[];
}

export function LiveTable({ rows }: Props) {
  return (
    <section className="card">
      <h2>Live Cluster Presence</h2>
      <table>
        <thead>
          <tr>
            <th>Apprentice</th>
            <th>Machine</th>
            <th>Last Seen</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.apprentice_id + row.machine_id}>
              <td>{row.apprentice_id}</td>
              <td>{row.machine_id}</td>
              <td>{new Date(row.last_seen).toLocaleString()}</td>
            </tr>
          ))}
          {rows.length === 0 && (
            <tr>
              <td colSpan={3}>No active apprentices.</td>
            </tr>
          )}
        </tbody>
      </table>
    </section>
  );
}
