$ErrorActionPreference = "Stop"

$LauncherRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $LauncherRoot
$LauncherSourceRoot = Join-Path $LauncherRoot "src"
$ServerSourceRoot = Join-Path $LauncherRoot "server-wrapper"
$ServerBundleRoot = Join-Path $LauncherRoot "server-bundle"
$WebDistRoot = Join-Path $LauncherRoot "web-dist"
$LauncherOutPath = Join-Path $LauncherRoot "AVmatrixLauncher.exe"
$ServerOutPath = Join-Path $ServerBundleRoot "avmatrix-server.exe"
$CliRoot = Join-Path $RepoRoot "avmatrix"
$WebRoot = Join-Path $RepoRoot "avmatrix-web"
$WebBuildRoot = Join-Path $WebRoot "dist"

function Assert-Command($Name) {
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "$Name is required to build the packaged launcher."
  }
}

Assert-Command "go"
Assert-Command "npm"
Assert-Command "node"

Push-Location $CliRoot
try {
  npm run build
} finally {
  Pop-Location
}

Push-Location $WebRoot
try {
  npm run build
} finally {
  Pop-Location
}

Push-Location $LauncherSourceRoot
try {
  go build -ldflags="-s -w -H=windowsgui" -o $LauncherOutPath .
} finally {
  Pop-Location
}

New-Item -ItemType Directory -Path $ServerBundleRoot -Force | Out-Null

Push-Location $ServerSourceRoot
try {
  go build -ldflags="-s -w -H=windowsgui" -o $ServerOutPath .
} finally {
  Pop-Location
}

$NodePath = (Get-Command node).Source
Copy-Item -LiteralPath $NodePath -Destination (Join-Path $ServerBundleRoot "node.exe") -Force

if (Test-Path -LiteralPath $WebDistRoot) {
  Remove-Item -LiteralPath $WebDistRoot -Recurse -Force
}
Copy-Item -LiteralPath $WebBuildRoot -Destination $WebDistRoot -Recurse -Force

& $LauncherOutPath register
