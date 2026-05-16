param(
  [string]$Version = "auto",
  [string]$OutputRoot = ".tmp\ladybug-native"
)

$ErrorActionPreference = "Stop"

$ScriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptRoot

function Resolve-RepoPath($Path) {
  if ([System.IO.Path]::IsPathRooted($Path)) {
    return [System.IO.Path]::GetFullPath($Path)
  }
  return [System.IO.Path]::GetFullPath((Join-Path $RepoRoot $Path))
}

function Resolve-LatestVersionTag($OutputRootFull) {
  $CachePath = Join-Path $OutputRootFull "latest-release.json"
  New-Item -ItemType Directory -Path $OutputRootFull -Force | Out-Null
  $TodayUtc = [DateTime]::UtcNow.ToString("yyyy-MM-dd")
  if (Test-Path -LiteralPath $CachePath) {
    try {
      $Cached = Get-Content -LiteralPath $CachePath -Raw | ConvertFrom-Json
      if ($Cached.checkedDateUtc -eq $TodayUtc -and $Cached.tag_name) {
        return [string]$Cached.tag_name
      }
    } catch {
      # Ignore invalid cache content and refresh from GitHub.
    }
  }

  $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/LadybugDB/ladybug/releases/latest" -Headers @{
    "Accept" = "application/vnd.github+json"
    "User-Agent" = "avmatrix-go-native-bootstrap"
  }
  if (-not $Release.tag_name) {
    throw "Could not resolve latest LadybugDB release tag from GitHub."
  }

  $Payload = [ordered]@{
    tag_name = [string]$Release.tag_name
    checkedDateUtc = $TodayUtc
    checkedAtUtc = [DateTime]::UtcNow.ToString("o")
  }
  $Payload | ConvertTo-Json | Set-Content -LiteralPath $CachePath -Encoding UTF8
  return [string]$Release.tag_name
}

function Assert-ChildPath($Parent, $Child) {
  $ParentFull = [System.IO.Path]::GetFullPath($Parent).TrimEnd([System.IO.Path]::DirectorySeparatorChar)
  $ChildFull = [System.IO.Path]::GetFullPath($Child).TrimEnd([System.IO.Path]::DirectorySeparatorChar)
  if (-not $ChildFull.StartsWith($ParentFull, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to write outside native output root: $ChildFull"
  }
}

function Test-NativeFiles($Path) {
  return (Test-Path -LiteralPath (Join-Path $Path "lbug.h")) -and
    (Test-Path -LiteralPath (Join-Path $Path "lbug_shared.lib")) -and
    (Test-Path -LiteralPath (Join-Path $Path "lbug_shared.dll"))
}

if ($env:PROCESSOR_ARCHITECTURE -notin @("AMD64", "x86_64")) {
  throw "LadybugDB Windows native runtime is wired for x86_64; current architecture is $env:PROCESSOR_ARCHITECTURE."
}

$OutputRootFull = Resolve-RepoPath $OutputRoot
$VersionTag = if ($Version -eq "auto" -or [string]::IsNullOrWhiteSpace($Version)) {
  Resolve-LatestVersionTag $OutputRootFull
} elseif ($Version.StartsWith("v")) {
  $Version
} else {
  "v$Version"
}
$VersionNumber = $VersionTag.TrimStart("v")
$NativeDir = Join-Path $OutputRootFull (Join-Path $VersionTag "windows-x86_64")
if (Test-NativeFiles $NativeDir) {
  Write-Output $NativeDir
  exit 0
}

New-Item -ItemType Directory -Path $OutputRootFull -Force | Out-Null
$DownloadsDir = Join-Path $OutputRootFull "downloads"
$ExtractRoot = Join-Path $OutputRootFull "extract"
New-Item -ItemType Directory -Path $DownloadsDir -Force | Out-Null
New-Item -ItemType Directory -Path $ExtractRoot -Force | Out-Null

$ArchivePath = Join-Path $DownloadsDir "liblbug-windows-x86_64-$VersionNumber.zip"
if (-not (Test-Path -LiteralPath $ArchivePath)) {
  $Url = "https://github.com/LadybugDB/ladybug/releases/download/$VersionTag/liblbug-windows-x86_64.zip"
  Invoke-WebRequest -Uri $Url -OutFile $ArchivePath
}

$TempExtract = Join-Path $ExtractRoot "windows-x86_64-$VersionNumber"
Assert-ChildPath $OutputRootFull $TempExtract
if (Test-Path -LiteralPath $TempExtract) {
  Remove-Item -LiteralPath $TempExtract -Recurse -Force
}
New-Item -ItemType Directory -Path $TempExtract -Force | Out-Null
Expand-Archive -LiteralPath $ArchivePath -DestinationPath $TempExtract -Force

$Header = Get-ChildItem -LiteralPath $TempExtract -Recurse -Filter "lbug.h" | Select-Object -First 1
if (-not $Header) {
  throw "Downloaded LadybugDB archive did not contain lbug.h."
}
$SourceDir = $Header.DirectoryName

Assert-ChildPath $OutputRootFull $NativeDir
if (Test-Path -LiteralPath $NativeDir) {
  Remove-Item -LiteralPath $NativeDir -Recurse -Force
}
New-Item -ItemType Directory -Path $NativeDir -Force | Out-Null
Copy-Item -Path (Join-Path $SourceDir "*") -Destination $NativeDir -Recurse -Force

if (-not (Test-NativeFiles $NativeDir)) {
  throw "LadybugDB native Windows runtime is incomplete in $NativeDir."
}

Write-Output $NativeDir
