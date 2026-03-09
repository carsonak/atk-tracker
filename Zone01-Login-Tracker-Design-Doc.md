# Zone01 Apprentice Activity Tracker

## 1. Overview and Goals

The Zone01 Apprentice Activity Tracker is a client-server system designed to
monitor and log the physical presence and active keyboard/mouse engagement of
apprentices at a Zone01 cluster.
By translating raw hardware interactions into quantifiable "active time",
the system aims to provide an accurate estimation of actual hours spent working
at the cluster, rather than just relying on standard login/logout sheets.

### Primary Goals

* Accurately measure *"active"* keyboard/mouse time for each apprentice.
* Securely and efficiently transmit this data to a centralized database.
* Provide a reliable way to aggregate and query total working hours per apprentice
over specific timeframes (daily, weekly, monthly).
* Ensure the system runs unobtrusively in the background without degrading machine
performance.

## 2. Non-Goals

* **Keylogging/Spyware:** The system will ***never*** record which specific keys are pressed,
what is clicked, or what is on the screen. It only records binary activity (active vs. inactive).
* **Grading/Evaluation:** This system tracks presence, not the quality or output of the work.
It is an attendance tool, not a performance metric tool.
* **Remote Tracking:** The system is designed strictly for local Zone01 cluster machines,
not for personal laptops at home.

## 3. Assumptions & Prerequisites

For this system to function as designed, the following conditions must be met:

* **Unique OS Accounts:** Apprentices log into the cluster machines using unique,
identifiable credentials.
* **OS Permissions:** The tracking daemon has the necessary OS-level permissions to
hook into global Human Interface Device (**HID**) events without requiring root access
for the user, or it runs as a privileged background service.
* **Network Connectivity:** While the system handles offline buffering, machines must
have periodic internet/intranet access to sync data with the central server.
* **System Time Sync:** All cluster machines use NTP (Network Time Protocol) to ensure
timestamps are accurate across the network.

## 4. Domain Model

The data architecture relies on a few core entities to connect users, their machines
and their activity.

* **Apprentice:** Represents the user `{ID, Name, Cohort}`.
* **Machine:** Represents the physical cluster computer
`{MAC Address, Hostname, Cluster Zone}`.
* **Session:** Represents a single continuous login period on a specific machine
`{Session ID, Apprentice ID, Machine ID, Login Time, Logout Time}`.
* **Heartbeat (Log):** Represents a slice of time within a session
`{Heartbeat ID, Session ID, Timestamp, Active Seconds}`.

## 5. Core System Logic

### Processing HID Events & "Active Time"

Instead of logging every keystroke, the local agent divides time into
**1-second buckets**.

* If a mouse movement, click, or keystroke is detected during a 1-second bucket,
that second is marked as `Active: True`.
* If no event occurs, it is `Active: False`.

### The "Heartbeat"

To minimize network traffic, the client agent aggregates these 1-second buckets
over a **5-minute window** (300 seconds).

* At the end of the 5 minutes, the agent sends a JSON payload (the "Heartbeat") to
the server stating: *"In the last 5 minutes, User X was active for 215 seconds."*
* If the user walks away, the agent eventually sends a heartbeat with `0` active
seconds, signalling an idle state.

### Handling Machine Switches

If an apprentice leaves Machine A and logs into Machine B:

1. Machine A's OS triggers a logout/lock event. The agent immediately flushes its
current buffer, sends a final heartbeat, and closes the `Session`.
2. Machine B creates a new `Session` and begins a new heartbeat cycle.
3. *Edge Case (Abrupt Power Loss on Machine A):* If Machine B starts a new session
while Machine A's session is technically still "open" on the server, the server will
automatically terminate Machine A's session based on a missed heartbeat timeout
(e.g., no heartbeat received for 15 minutes).

## 6. Responsibility Segregation

### Client Agent (Local Daemon)

* **Detect:** Listen for OS login/logout events.
* **Monitor:** Read standard HID event files (e.g., `/dev/input` on Linux) securely.
* **Aggregate:** Run the 1-second bucket logic and sum the active seconds.
* **Buffer & Transmit:** Send the 5-minute heartbeat to the API. If the network is down,
store heartbeats in a local SQLite database or JSON file until connectivity is restored.

### API Server (Backend)

* **Authenticate:** Verify incoming heartbeats belong to valid, active apprentices.
* **Validate:** Ensure timestamps make sense (reject heartbeats from the future or
impossibly large active durations).
* **Persist:** Write the heartbeats to the database.

### Database

* **Store:** Safely house the time-series heartbeat data.
* **Aggregate:** Provide fast querying for the UI/Admins
(e.g., "Sum all active seconds for Apprentice Y in March").

## 7. System Flow: How It Works

1. **Login:** Apprentice logs into a cluster machine. The OS launches the Client
Agent daemon.
2. **Session Init:** The agent pings the API: `POST /sessions` to get a new
`session_id`.
3. **Work Phase:** Apprentice writes code. The agent counts active seconds locally.
4. **Sync Phase:** Every 5 minutes, the agent sends `POST /heartbeats`.
5. **Idle Phase:** Apprentice goes to lunch. The OS locks the screen, or the agent
detects 0 active seconds and reports it.
6. **Logout:** Apprentice logs out. The agent catches the SIGTERM signal, flushes
the final active seconds, sends a `PUT /sessions/{id}/end`, and shuts down.

## 8. Failure Cases & Mitigation

| Failure Scenario | Mitigation Strategy |
| --- | --- |
| **Network Disconnect** | Agent saves heartbeats to a local queue. Upon reconnection, it bulk-sends historical heartbeats using their original timestamps. |
| **Agent Crash** | The OS process manager (e.g., `systemd`) is configured to automatically restart the daemon if it fails. |
| **Apprentice Kills Process** | Run the daemon under a dedicated service account, preventing the standard user from terminating the process via a task manager. |
| **Database Downtime** | API server queues incoming heartbeats in a fast, in-memory datastore until the primary DB is restored. |

## 9. Design Decisions & Trade-offs

### Fat Client vs. Thin Client

> * **Decision:** Fat client, local aggregation of data into "heart beats".
> * **Reasoning:** Drastically reduces network traffic and database writes,
> while completely eliminating the privacy risk of transmitting raw keystroke
> patterns over the network.

### Heartbeat vs. Real-time WebSockets

> * **Decision:** 5-minute REST API heartbeats.
> * **Reasoning:** WebSockets require persistent open connections,
> which can be unstable across hundreds of cluster machines and require heavy
> server resources. REST is stateless, scalable, and handles offline buffering naturally.

## 10. Future Plans

* **Idle Warning Notifications:** If the agent detects 15 minutes of 0 active seconds,
pop up an OS notification reminding the user to lock their screen or log out.
* **Admin Dashboard:** A web interface for Zone01 staff to view real-time cluster occupancy
and generate attendance reports.
* **IDE Integration:** Instead of purely OS-level tracking, a future VS Code extension
could track "active typing in editor" for an even more accurate metric of coding time vs.
web browsing time.
