# ATK (At The Keyboard) Tracker

## 1. Overview and Goals

The ATK (At The Keyboard) Tracker is a client-server system designed to monitor and log the
physical presence and active keyboard/mouse engagement of apprentices at a Zone01 cluster.
By translating raw hardware interactions into quantifiable "active time", the system aims to
provide an accurate estimation of actual hours spent working at the cluster, rather than just
relying on standard login/logout sheets.

### Primary Goals

* Accurately measure *"active"* keyboard/mouse time for each apprentice.
* Securely and efficiently transmit this data to a centralized database.
* Provide a reliable, highly scalable way to aggregate and query total working hours per
apprentice over specific timeframes (daily, weekly, monthly).
* Ensure the system runs unobtrusively in the background without degrading machine performance.
* Remain cost-efficient by preventing long-term database snowballing.

### Non-Goals

* **Keylogging/Spyware:** The system will ***never*** record which specific keys are pressed,
what is clicked, or what is on the screen.
It only records binary activity (active vs. inactive).
* **Grading/Evaluation:** This system tracks presence, not the quality or output of the work. It is an attendance tool, not a performance metric tool.
* **Remote Tracking:** The system is designed strictly for local Zone01 cluster machines, not for personal laptops at home.

## 2. Assumptions & Prerequisites

For this system to function as designed, the following conditions must be met:

* **System-Level Execution:** The tracking daemon runs as a privileged `systemd` background
service (e.g., root or `input` group), ensuring it runs regardless of who logs in and cannot
be easily terminated by non-admin users.
* **Stateless Clients:** Cluster machines do not need an updated list of apprentices.
The OS handles LDAP/network authentication, and the backend API validates users dynamically.
* **Unique OS Accounts:** Apprentices log into the cluster machines using unique,
identifiable credentials.
* **Network Connectivity:** While the system handles offline buffering,
machines must have periodic intranet access to sync data.

## 3. Domain Model

The data architecture relies on a few core entities to connect users, their machines, and their activity.

* **Apprentice:** Represents the user `{ID, Name, Cohort}`.
* **Machine:** Represents the physical cluster computer `{MAC Address, Hostname, Cluster Zone}`.
* **Session:** Represents a single continuous login period on a specific machine `{Session ID, Apprentice ID, Machine ID, Login Time, Logout Time}`.
* **Heartbeat (Log):** Represents a slice of time within a session `{Heartbeat ID, Session ID, Timestamp, Active Seconds}`.

## 4. Core System Logic

### Startup State Reconciliation (Self-Healing)

If the daemon is updated or restarted by the OS, it wakes up "blind" to the current user.
Before initializing event listeners, the daemon queries the OS (via D-Bus `ListSessions`)
to check for currently active graphical/TTY users. If an apprentice is already logged in,
the daemon seamlessly adopts the session, requests a new ID from the server,
and resumes tracking.

### Hardware Filtering

To avoid logging false positives from accelerometers or ignoring non-standard peripherals,
the daemon dynamically scans `/dev/input/event*` files on startup.
It uses `ioctl` system calls to request the hardware's capability bitmask, securely attaching
listeners *only* to devices that support `EV_KEY` (Keys/Buttons) or `EV_REL` (Mouse Movement).
Mics and cameras are physically isolated by the Linux kernel in different subsystems and cannot
be accessed.

### Processing HID Events & "Active Time"

Instead of logging every keystroke, the local agent divides time into **1-second buckets**.

* If an event is detected during a 1-second bucket, that second is marked as `Active: True`.
* If no event occurs, it is `Active: False`.

### The "Heartbeat"

To minimize network traffic, the client agent aggregates these 1-second buckets over a
**5-minute window** (300 seconds).

* At the end of the 5 minutes, the agent sends a JSON payload to the server stating:
*"In the last 5 minutes, User X was active for 215 seconds."*
* If the user walks away, the agent sends a heartbeat with `0` active seconds, signalling an idle state.

### Handling Overlapping Sessions ("The Two Machine Bounce")

If an apprentice leaves Machine A and logs into Machine B without logging out,
the system permits **overlapping sessions**.

* Both daemons will send heartbeats for their respective sessions.
* The backend database handles the complexity by flattening and merging overlapping
timestamps during query time, ensuring that the apprentice is never double-credited
for hours, while preventing frustrating session dropouts.

## 5. Responsibility Segregation

### Client Agent (Local Go Daemon)

* **Detect:** Listen for OS login/logout/suspend events via D-Bus.
* **Monitor:** Read standard HID event files securely.
* **Aggregate:** Run the 1-second bucket logic and sum the active seconds.
* **Buffer & Transmit:** Send the 5-minute heartbeat to the API.
If the network is down or a session is invalidated, request a new session,
buffer locally, and retry.

### API Server (Go Backend)

* **Authenticate:** Verify incoming heartbeats belong to valid apprentices.
* **Live State:** Maintain an in-memory or lightweight map of `last_seen` timestamps
to power real-time dashboard occupancy views.
* **Persist:** Write the heartbeats to the database.

### Database (Data Rollup Architecture)

To prevent millions of rows from slowing down the server over a 1.5-year cohort tenure,
the database uses a Rollup strategy:

* **`raw_heartbeats` Table:** Stores the exact 5-minute interval payloads.
A TTL script automatically deletes rows older than 30 days.
* **`daily_summaries` Table:** A permanent archive.
A nightly cron job sums the `raw_heartbeats` for the day and writes a single row per user
(e.g., `User X, Date Y, Total_Active: 420 mins`).
* **Result:** Millisecond queries for long-term frontend graphs, with zero data snowballing.

## 6. System Flow: How It Works

1. **Boot:** Machine powers on. Daemon starts, checks for active users, and awaits a login.
2. **Login:** Apprentice logs in. The OS triggers a D-Bus event.
3. **Session Init:** The agent pings the API: `POST /sessions` to get a new `session_id`.
4. **Work Phase:** Apprentice works. The agent counts active seconds locally.
5. **Sync Phase:** Every 5 minutes, the agent sends `POST /heartbeats`.
Server updates the user's `last_seen` status for the live dashboard.
6. **Idle Phase:** Apprentice steps away. The agent detects 0 active seconds and reports it.
7. **Logout:** Apprentice logs out. The agent flushes the final buffer, sends a `PUT /sessions/{id}/end`, and returns to standby mode.

## 7. Failure Cases & Mitigation

| Failure Scenario | Mitigation Strategy |
| --- | --- |
| **Network Disconnect** | Agent saves heartbeats to a local queue. Upon reconnection, it bulk-sends historical heartbeats. |
| **Agent Crash/Killed** | `systemd` automatically restarts the daemon (`Restart=always`). On boot, the daemon uses D-Bus to re-adopt the existing session. |
| **Apprentice Malice** | If the user forcefully uninstalls the tracker via `sudo`, no heartbeats are sent, and they receive no hours. Incentive is aligned with system compliance. |
| **Database Downtime** | API server queues incoming heartbeats in memory until the DB is restored. |

## 8. Design Decisions & Trade-offs

* **Language Choice (Go):** Selected for both the client and backend. Go compiles to a single,
dependency-free binary for the client, minimizing cluster administration overhead,
while offering massive concurrent request scaling for the backend.
* **Fat Client vs. Thin Client:** Fat client (local aggregation) chosen to reduce network
traffic to ~0.66 requests per second (even during high capacity) and entirely eliminate
privacy risks associated with transmitting raw keystrokes.
* **Data Rollup vs. Infinite Storage:** Selected to ensure the system remains extremely cheap
to host. The trade-off is losing minute-by-minute granularity for records older than 30 days,
which is acceptable for attendance tracking.

## 9. Future Plans

* **Idle Warning Notifications:** If the agent detects 15 minutes of 0 active seconds,
pop up an OS notification reminding the user to lock their screen or log out.
* **IDE Integration:** A future VS Code extension to track "active typing in editor" for
an even more granular metric of coding vs. browsing time.
