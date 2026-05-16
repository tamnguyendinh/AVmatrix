package httpapi

import (
	"errors"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

var errFolderPickerUnsupported = errors.New("local folder picker is not supported on this operating system; paste the absolute path manually")

var pickLocalFolderFunc = pickLocalFolder

func (s Server) handleLocalFolderPicker(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	path, err := pickLocalFolderFunc()
	if err != nil {
		if errors.Is(err, errFolderPickerUnsupported) {
			writeError(w, http.StatusNotImplemented, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cancelled := path == ""
	writeJSON(w, http.StatusOK, map[string]any{"path": emptyStringToNil(path), "cancelled": cancelled})
}

func pickLocalFolder() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return pickWindowsFolder()
	case "darwin":
		return pickCommandFolder("osascript", "-e", `POSIX path of (choose folder with prompt "Choose repository folder")`)
	case "linux":
		if path, err := pickCommandFolder("zenity", "--file-selection", "--directory", "--title=Choose repository folder"); err == nil || path != "" {
			return path, err
		}
		return pickCommandFolder("kdialog", "--getexistingdirectory", ".", "Choose repository folder")
	default:
		return "", errFolderPickerUnsupported
	}
}

func pickWindowsFolder() (string, error) {
	script := `
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.Application]::EnableVisualStyles()
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = 'Choose repository folder'
$dialog.ShowNewFolderButton = $false
$result = $dialog.ShowDialog()
if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
  [Console]::OutputEncoding = [System.Text.Encoding]::UTF8
  Write-Output $dialog.SelectedPath
  exit 0
}
exit 2
`
	return pickCommandFolder("powershell.exe", "-NoProfile", "-STA", "-ExecutionPolicy", "Bypass", "-Command", script)
}

func pickCommandFolder(name string, args ...string) (string, error) {
	output, err := exec.Command(name, args...).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 1 || exitErr.ExitCode() == 2) {
			return "", nil
		}
		if errors.Is(err, exec.ErrNotFound) {
			return "", errFolderPickerUnsupported
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func emptyStringToNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}
