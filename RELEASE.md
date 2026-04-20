# Release engineering for Grove

## How to publish a release

1. Update versioned code if needed.
2. Run local verification:
   - go test ./...
   - go build ./...
3. Create and push a semver tag:
   - git tag -a v0.1.1 -m "Grove v0.1.1"
   - git push origin v0.1.1
4. GitHub Actions will build and publish release artifacts.

## User install paths

- Go users:
  - go install github.com/saddatahmad19/grove/cmd/grove@latest
- Binary users:
  - download from GitHub Releases
  - extract and place `grove` on PATH

## Notes

- The module path is github.com/saddatahmad19/grove.
- The repo is public and tag-based release automation is enabled.
