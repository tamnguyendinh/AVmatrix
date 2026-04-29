import { execFile } from 'child_process';
import { promisify } from 'util';

const execFileAsync = promisify(execFile);

export class UnsupportedFolderPickerError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'UnsupportedFolderPickerError';
  }
}

const normalizeSelectedPath = (stdout: string): string | null => {
  const selectedPath = stdout.trim();
  return selectedPath.length > 0 ? selectedPath : null;
};

const isCancelExit = (err: unknown): boolean => {
  if (!err || typeof err !== 'object') return false;
  const code = String((err as NodeJS.ErrnoException & { code?: string | number }).code ?? '');
  return code === '1' || code === '2';
};

const pickWindowsFolder = async (): Promise<string | null> => {
  const script = `
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
  `;

  try {
    const { stdout } = await execFileAsync('powershell.exe', [
      '-NoProfile',
      '-STA',
      '-ExecutionPolicy',
      'Bypass',
      '-Command',
      script,
    ]);
    return normalizeSelectedPath(stdout);
  } catch (err) {
    if (isCancelExit(err)) return null;
    throw err;
  }
};

const pickMacFolder = async (): Promise<string | null> => {
  try {
    const { stdout } = await execFileAsync('osascript', [
      '-e',
      'POSIX path of (choose folder with prompt "Choose repository folder")',
    ]);
    return normalizeSelectedPath(stdout);
  } catch (err) {
    if (isCancelExit(err)) return null;
    throw err;
  }
};

const pickLinuxFolder = async (): Promise<string | null> => {
  try {
    const { stdout } = await execFileAsync('zenity', [
      '--file-selection',
      '--directory',
      '--title=Choose repository folder',
    ]);
    return normalizeSelectedPath(stdout);
  } catch (zenityErr) {
    if (isCancelExit(zenityErr)) return null;

    try {
      const { stdout } = await execFileAsync('kdialog', [
        '--getexistingdirectory',
        '.',
        'Choose repository folder',
      ]);
      return normalizeSelectedPath(stdout);
    } catch (kdialogErr) {
      if (isCancelExit(kdialogErr)) return null;
      throw new UnsupportedFolderPickerError(
        'No local folder picker is available. Install zenity or kdialog, or paste the absolute path manually.',
      );
    }
  }
};

export const pickLocalFolder = async (): Promise<string | null> => {
  if (process.platform === 'win32') return pickWindowsFolder();
  if (process.platform === 'darwin') return pickMacFolder();
  if (process.platform === 'linux') return pickLinuxFolder();

  throw new UnsupportedFolderPickerError(
    'Local folder picker is not supported on this operating system. Paste the absolute path manually.',
  );
};
