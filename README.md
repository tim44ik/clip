# clip

**clip** is a utility for running single‑task and multi‑task scenarios with a graphical user interface (GUI) built on [Fyne](https://fyne.io/).  
It allows you to load, edit, and execute modules (scripts written in an internal scripting language), control concurrency, save/load encrypted configurations, and generate reports.

## Core Idea

Clip helps you automate sequences of actions, analyse data, and produce reports.  
You create a **scenario** that consists of a **main module** and **child modules**, each containing code in the embedded scripting language.

Modules are executed in the order defined by `queue()`. The **threads** setting controls how many modules from the same queue (group) can run **in parallel**.  
This allows you to fine‑tune concurrency while preserving the overall execution order.

Key features:

- Run scripts with variables, loops, conditionals, and built‑in functions (CVE/CPE database queries, external command execution, string processing).
- Parallel execution with configurable thread limits.
- Interactive GUI to manage scenarios (load, save, edit, start, interrupt).
- Generate reports (e.g., PDF) from execution results.
- Encrypted configuration files.
- **Designed to be run inside Docker** – the application always works with a pre‑filled PostgreSQL database (CVE/CPE data) and a full Kali Linux toolchain.

## Building & Running

Clip is meant to be used as a **Docker‑based solution**.  
The repository provides `Dockerfile.db`, `Dockerfile.app`, and `docker-compose.yml` to launch a self‑contained environment with Kali Linux, all necessary CLI tools, and a pre‑filled vulnerability database.

### Run with Docker

```bash
docker compose up -d
```

After starting, open your browser at `http://localhost:6080/vnc.html` – you will see the clip GUI.

There is **no native build outside Docker** – the architecture relies on the database and the Kali environment being present.

It also creates **/shared** folder in working directory so you can share config files and run external apps in the environment.

## User Interface

### Top Panel

#### Folder Icon Menu

- **Load Script** – Open a directory chooser to import scripts.
- **Load** – Load a scenario configuration (JSON format; encrypted files are supported).
- **Save** – Save the current scenario. First save asks for format, encryption, and password; subsequent saves overwrite the existing file.
- **Save As** – Save the scenario to a new location.

#### Scenario Control Button

- **Start** – Begin execution.
- **Interrupt** – Stop execution.

#### Language Button

Opens the language selection window (i18n support).

#### Exit Button

Closes the application.

### Central Panel

Split into two sections:

- **Module list** (left) – shows all modules with their names.
- **Input/Output area** (right) – displays the output of the selected module.

### Lower Panel

- **Threads Number** – Maximum number of goroutines (modules) that run **in parallel** inside a single queue.
- **View Full Output** – Opens a separate window showing the complete output captured at the moment the button is clicked (no live updates). This works around Fyne’s performance issues when rendering long logs – the main output view is limited to 14 lines.

### Module Actions (context menu or buttons)

- **Edit** – Rename the module.
- **Delete** – Remove the module from the scenario.

### Add Module

- **Add Module** – Opens a creation window. After saving, the user returns to the new module’s editing screen.

## Scenario Format

Scenarios are saved as JSON according to the following structure:

```go
type Module struct {
    Name    string `json:"name"`
    Content string `json:"content"`
    Output  string `json:"-"` // not stored, only runtime
}

type ClipModules struct {
    MainModule   *Module   `json:"mainModule"`
    ChildModules []*Module `json:"childModules"`
    CurrentLang  string    `json:"currentLang"`
}
```

Example:

```json
{
  "mainModule": {
    "name": "main",
    "content": "%a = \"hello\"\nprint(%a)"
  },
  "childModules": [
    {
      "name": "child1",
      "content": "print(\"child1 output\")"
    }
  ],
  "currentLang": "en"
}
```

- The **MainModule** is executed first, followed by the **ChildModules** in the order they appear in the array.
- The `Output` field is used at runtime and never persisted to the JSON file.

## Scripting Language

The embedded interpreter supports:

- Variables prefixed with `%` (numbers, strings, arrays).
- Arithmetic: `+ - * / %`.
- Comparisons: `== != < > <= >=`.
- Logical operators: `and`, `or`, `not`.
- Conditional: `if ... then ... else ... end`.
- Loop: `for init; cond; post do ... end` with `break` and `continue`.
- Built‑in functions:  
  `print`, `len`, `append`, `split`, `fields`, `contains`, `replace`,  `str`, `int`
  `run` (execute shell commands with persistent context),
  `runIsolated` (execute shell commands without persistent context)  
  `process` (query the CVE/CPE database),  
  `report` (generate a report).
- Modules run in separate environments but can share global state (report, database, main module content).

[full language documentation](docs/lang_doc.md)

## License

[MIT](LICENSE)