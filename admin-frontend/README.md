# ATK Admin Frontend

React + TypeScript dashboard for operations and reporting.

## Views

- Live View: Current presence list from GET /live.
- Historical View: Raw heartbeat bars + daily summary bars from GET /stats.

## Development

1. Install dependencies:

   npm install

2. Configure API endpoint:

   export VITE_API_URL=http://127.0.0.1:8080

3. Start dev server:

   npm run dev

4. Build production assets:

   npm run build

5. Preview production build:

   npm run preview

## Notes

- Frontend expects apprentice_id values that exist in server data.
- For demo data, use values like demo-anna, demo-bao, demo-caro.
