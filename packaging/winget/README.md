# Winget Package

This directory contains the Winget manifests for gro (google-readonly CLI).

## Automated Publishing

Winget packages are automatically submitted when a new release is created.
The GitHub Actions workflow:

1. Downloads release checksums
2. Updates version and checksums in manifests
3. Submits a PR to microsoft/winget-pkgs

## Manual Publishing

If you need to publish manually:

1. Update version in all three manifest files
2. Update checksums in `OpenCLICollective.gro.installer.yaml`
3. Validate: `winget validate --manifest .`
4. Submit using wingetcreate or manually create a PR to microsoft/winget-pkgs

## Installation

```powershell
winget install OpenCLICollective.gro
```

## Manifest Files

- `OpenCLICollective.gro.yaml` - Version manifest
- `OpenCLICollective.gro.installer.yaml` - Installer manifest with URLs and checksums
- `OpenCLICollective.gro.locale.en-US.yaml` - Localized package metadata
