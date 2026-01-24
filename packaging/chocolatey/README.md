# Chocolatey Package

This directory contains the Chocolatey package for gro (google-readonly CLI).

## Automated Publishing

Chocolatey packages are automatically published when a new release is created.
The GitHub Actions workflow:

1. Downloads release checksums
2. Injects checksums into `chocolateyInstall.ps1`
3. Updates the version in `google-readonly.nuspec`
4. Packs and pushes to Chocolatey.org

## Manual Publishing

If you need to publish manually:

1. Update version in `google-readonly.nuspec`
2. Update checksums in `tools/chocolateyInstall.ps1`
3. Pack: `choco pack`
4. Push: `choco push google-readonly.<version>.nupkg --source https://push.chocolatey.org/ --key YOUR_API_KEY`

## Installation

```powershell
choco install google-readonly
```

## Package Structure

- `google-readonly.nuspec` - Package metadata
- `tools/chocolateyInstall.ps1` - Installation script
- `tools/chocolateyUninstall.ps1` - Uninstallation script
