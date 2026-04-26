# clip
an utility to run multi and single task scenarios in cli

## Interface

The top panel contains four buttons, each of which opens a dropdown menu.

### Folder Button Menu

The button with the folder icon opens a menu with the following options:

* **Load Script** — opens a folder selection dialog for choosing scripts to be imported into the application.
* **Load** — opens a file system dialog with a filter for configuration files (currently `.json`, but more formats can be added). If the configuration is encrypted, a password input dialog is displayed.
* **Load in New Window** — performs the same actions but loads the profile in a new window.
* **Save** — opens dialogs for selecting file format, encryption type, and password (if needed). If the profile was previously saved or loaded, the file is overwritten; otherwise, a file dialog is shown.
* **Save As** — opens a file system dialog to save the profile to a new location.

### Scenario Control

The second button allows you to:

* Start a scenario
* Interrupt a scenario
* Interrupt a scenario and generate a report

**Interrupt Scenario and Generate Report** can also create a PDF from a previous run without executing the scenario again.

### Language

The third button opens the language selection window.

### Exit

The fourth button closes the application window.

---

### Central Panel

Central panel is divided in two:
* Modules list on the left side
* Input and output entry
---

### Lower Panel

Checkboxes in the lower panel control report generation and output processing:

* If enabled on the main screen, they apply to all modules
* Disabling them affects all modules accordingly

### Generate Report

* After execution, opens a report format selection (currently only PDF)
* After execution, opens a file saving dialog

### Process Output

* Disabled if **Generate Report** is not selected
* If enabled:

  * After execution, a database selection window appears
  * The program performs a search based on CVE and product+version patterns
  * Information is retrieved from the selected database using regex matching

---

### Threads

**Threads Number** defines how many goroutines (modules) run simultaneously.
Execution order defined by `queue()` is preserved.

---

### Output

**View Full Output** opens a window displaying the complete output at the moment it is created (no live updates).
This feature was introduced to work around Fyne’s lack of scroll tracking. In earlier versions, the UI experienced lag because it rendered all output, including content outside the visible area.

To address this, the main output view was limited to 14 lines, and the **View Full Output** option was added.

---

### Module Actions

* **Edit** — opens a window to rename the module
* **Delete** — removes the module and returns to the main screen

---

### Add Module

**Add Module** opens a creation window and returns user to the new module screen after saving.

