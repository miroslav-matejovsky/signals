# Create and push a Go module tag, then publish its GitHub release.
#
# Usage: task release -- 0.1.0   (or v0.1.0)
#
# The tag supports go install; the GitHub release shows the version and generated notes.
# Preconditions, checked in order, each failing fast:
#   - version is valid semver (optional leading "v", optional pre-release / build suffix)
#   - current branch is main
#   - working tree has no uncommitted or untracked changes
#   - GitHub CLI is installed and authenticated
#   - local main is identical to origin/main (nothing unpushed, nothing unpulled)
#   - tag does not exist locally or on origin
# Then an annotated tag is created on HEAD, pushed to origin, and published on GitHub.
# Semver prereleases are marked as prereleases on GitHub.

param(
  [Parameter(Mandatory = $true)]
  [string]$Version
)

$ErrorActionPreference = "Stop"

function Invoke-Git {
  $out = & git @args
  if ($LASTEXITCODE -ne 0) { throw "git $($args -join ' ') failed" }
  return $out
}

$Version = $Version.Trim().Trim('"', "'")
if (-not $Version.StartsWith("v")) { $Version = "v$Version" }

$semver = '^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
if ($Version -notmatch $semver) { throw "invalid version '$Version', expected semver like 0.1.0 or v0.1.0" }

$root = Invoke-Git rev-parse --show-toplevel
Set-Location $root

$branch = Invoke-Git rev-parse --abbrev-ref HEAD
if ($branch -ne "main") { throw "must be on main, current branch is '$branch'" }

$status = Invoke-Git status --porcelain
if ($status) { throw "working tree has uncommitted changes:`n$($status -join "`n")" }

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
  throw "GitHub CLI (gh) is required; install it and run 'gh auth login'"
}
& gh auth status
if ($LASTEXITCODE -ne 0) { throw "GitHub authentication failed; run 'gh auth login'" }

$origin = Invoke-Git remote get-url --push origin

Invoke-Git fetch origin main --tags --quiet | Out-Null

$local = Invoke-Git rev-parse HEAD
$remote = Invoke-Git rev-parse origin/main
if ($local -ne $remote) {
  $ahead = Invoke-Git rev-list --count origin/main..HEAD
  $behind = Invoke-Git rev-list --count HEAD..origin/main
  throw "main differs from origin/main (ahead $ahead, behind $behind), push or pull first"
}

if (Invoke-Git tag --list $Version) { throw "tag $Version already exists locally" }
if (Invoke-Git ls-remote --tags origin "refs/tags/$Version") { throw "tag $Version already exists on origin" }

Invoke-Git tag -a $Version -m "Release $Version" | Out-Null
Invoke-Git push origin $Version --quiet | Out-Null

$releaseArgs = @("release", "create", $Version, "--repo", $origin, "--verify-tag", "--title", $Version, "--generate-notes")
if ($Version -match '^v\d+\.\d+\.\d+-') { $releaseArgs += "--prerelease" }
& gh @releaseArgs
if ($LASTEXITCODE -ne 0) {
  throw "GitHub release creation failed; tag $Version was pushed. Check GitHub for an existing release, then retry if needed: gh $($releaseArgs -join ' ')"
}

Write-Host "released $Version at $local"
