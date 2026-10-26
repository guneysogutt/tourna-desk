# 🏆 Tourna-Desk

**Tourna-Desk** is an on-premise, single-binary **Sports Tournament, Draw & Fixture Management Desk** built with a pure-Go backend, embedded SQLite database, and a modern TypeScript/React + Tailwind CSS interface.

It compiles into a standalone executable (`CGO_ENABLED=0`) that runs offline on Windows, macOS, and Linux with zero external dependencies.

---

## ✨ Key Features

- **🏀 Multi-Sport Scoring & Tiebreak Engine**
  - Built-in official federation rules for **Basketball, Volleyball, Football, Futsal, and Handball**.
  - Automatic head-to-head (*ikili/çoklu averaj*), set ratios (Volleyball), penalty shootout tiebreakers (Futsal), and official forfeit (*hükmen*) score & point deductions (e.g., Basketball forfeit loser receives `0 pts` instead of `1 pt`).
- **📋 Bulk Team Importer (`.xls` & `.xlsx`)**
  - Dual-engine spreadsheet parser supporting both modern `.xlsx` files and legacy BIFF8 `.xls` spreadsheets.
  - Dynamically detects header rows and columns, sanitizes team names, disambiguates duplicate names by region/district, and prevents duplicate registrations.
- **🎲 Official Rotation Key & Multi-Stage Draw Engine**
  - **Stage 1 Seeded Group Draw (*Seri Başı Kura*):** Isolates seeded teams at Slot 1 of each group and schedules matches using the official `(1-4, 2-3)` rotation key.
  - **Pot 1 & Pot 2 Progression (*Torba Sistemi*):** Automatically populates group winners (Pot 1) and runners-up (Pot 2) with drag-and-drop wildcard overrides and previous-group collision avoidance.
  - **Final Four & Dynamic $2^n$ Knockout Brackets:** Cross-matched Semi-Finals, 3rd Place, and Final matches, or full single-elimination trees (`4, 8, 16, 32, 64` slots) with balanced automatic **BYE (*BAY*)** advancement.
- **🏟️ Global Venue Management (*Merkezi Tesis Yönetimi*)**
  - Season-independent sports hall & field registry with multi-sport filtering across draw, fixture, and score modals.
- **📊 Styled Excel Exports**
  - One-click `.xlsx` export for group standings, weekly fixtures, and side-by-side visual knockout brackets.
- **🌐 Bilingual (TR / EN) & Dark Mode UI**
  - Instant Turkish / English switching, dark/light theme persistence, resizable & collapsible sidebar, and LAN access URL for mobile referee/scorer devices on the same Wi-Fi network.

---

## 📐 Official Sport Rules Matrix

| Sport | Win | Draw | Loss | Forfeit Score (*Hükmen*) | Primary Tiebreak Order |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Basketball** | `2 Pts` | — | `1 Pt` *(Forfeit: `0 Pts`)* | `20 - 0` | Points → H2H → Point Diff → Points Scored |
| **Volleyball** | `3-0/3-1`: `3 Pts`<br>`3-2`: `2 Pts` | — | `2-3`: `1 Pt`<br>`0-3/1-3`: `0 Pts` | `3 - 0` | Wins → Points → Set Ratio (`SW/SL`) → Point Ratio |
| **Futsal** | `3 Pts` | `1 Pt` *(+Penalties)* | `0 Pts` | `5 - 0` | Points → H2H / Penalties → Goal Diff → Goals For |
| **Football** | `3 Pts` | `1 Pt` | `0 Pts` | `3 - 0` | Points → H2H → Goal Diff → Goals For |
| **Handball** | `2 Pts` | `1 Pt` | `0 Pts` | `10 - 0` | Points → H2H → Goal Diff → Goals For |

---

## 🛠️ Tech Stack

- **Backend:** Go (`net/http`, `embed.FS`, `CGO_ENABLED=0`)
- **Database:** Pure-Go SQLite (`modernc.org/sqlite` in WAL mode)
- **Spreadsheet Engine:** `github.com/xuri/excelize/v2` (`.xlsx`) + pure-Go BIFF8/OLE2 reader via `github.com/richardlehane/mscfb` (`.xls`)
- **Frontend:** TypeScript, React 18, Vite, Tailwind CSS, `i18next`, `lucide-react`

---

## 🚀 Building & Running

### Prerequisites (For Building from Source)
- **Go** `1.22+`
- **Node.js** `18+` & `npm`

### 1. Build & Run Locally (macOS / Linux)
```bash
# Build frontend + embed into Go binary + launch on http://localhost:8080
make run
```

### 2. Cross-Compile Standalone Executables
```bash
# Build for current OS
make build

# Cross-compile standalone Windows executable (tourna-desk.exe)
make build-windows

# Cross-compile macOS Apple Silicon executable (tourna-desk-darwin-arm64)
make build-mac

# Cross-compile Linux amd64 executable (tourna-desk-linux-amd64)
make build-linux
```

On Windows, you can also build directly by running:
```bat
build.bat
```

### 3. Command-Line Flags
When running the compiled executable (`./tourna-desk` or `tourna-desk.exe`):
```bash
./tourna-desk -port 8080 -db tourna.db
```
| Flag | Default | Description |
| :--- | :--- | :--- |
| `-port` | `8080` | HTTP server port (can also be set via `PORT` env var) |
| `-db` | `tourna.db` | SQLite database file path (created automatically if missing) |
| `-no-browser` | `false` | Disable automatically opening the default browser on startup |

---

## 📂 Project Structure

```text
tourna-desk/
├── cmd/server/          # Main entrypoint & embedded static frontend (dist/)
├── internal/
│   ├── database/        # SQLite schema migrations & initial seed data
│   ├── engine/          # Round-robin, seeded draw, pots, knockout & scoring engines
│   │   └── scoring/     # Sport-specific standings & tiebreaker strategies
│   ├── handler/         # HTTP REST API handlers
│   ├── i18n/            # Server-side TR/EN export dictionary
│   ├── models/          # Domain models & DTOs
│   └── service/         # Tournament lifecycle, Excel export & .xls/.xlsx team import
├── web/                 # TypeScript + React + Tailwind CSS frontend
│   └── src/
│       ├── components/  # UI panels, bracket trees, modals & sport badges
│       ├── locales/     # Turkish (tr.json) & English (en.json) translations
│       └── utils/       # Custom confirm modal & UI utilities
├── Makefile             # Cross-platform build & test targets
└── build.bat            # Windows one-click build script
```

---

## 🧪 Running Tests

```bash
make test
# or
go test -v ./...
```

---

## 📄 License

Distributed under the MIT License. See [`LICENSE`](./LICENSE) for more information.
