# Keeping this fork on top of upstream Wings

This fork carries commits on top of [pelican-dev/wings](https://github.com/pelican-dev/wings).
Current base: `v1.0.0-beta27`.

| File | Change |
| --- | --- |
| `config/config_docker.go` | new `docker.devices` allowlist + `DeviceConfiguration` struct, additive |
| `environment/docker/devices.go` | new file, all passthrough logic |
| `environment/docker/devices_test.go` | new file |
| `environment/docker/container.go` | **one** call: `e.applyDevices(hostConf, evs)` |
| `environment/gpu.go` | new file, the `GPUStats` wire type |
| `environment/gpu_test.go` | new file |
| `environment/stats.go` | one field: `GPU *GPUStats` on `Stats` |
| `environment/docker/gpu_stats.go` | new file, the amdgpu sysfs collector |
| `environment/docker/gpu_stats_test.go` | new file |
| `environment/docker/stats.go` | one line: `GPU: e.gpuStats()` in the struct literal |
| `server/resources.go` | one line: `ru.GPU = nil` in `Reset()` |
| `README.md` | two new sections appended, not a full replace |

## Remotes

```bash
git remote -v
# origin    https://github.com/PerrysSpace/GWings   (fork)
# upstream  https://github.com/pelican-dev/wings.git (upstream)
```

## Pulling in a new upstream release
```bash
git fetch upstream --tags
TARGET=v1.0.0-betaXX   # pick a newer tag
git checkout -b "rebase-onto-$TARGET" main
git rebase "$TARGET"
```

### Expected conflicts

- [environment/docker/container.go](https://github.com/PerrysSpace/GWings/blob/main/environment/docker/container.go) — upstream moved code around the hostConf literal. Keep upstreams version, re-add `e.applyDevices(hostConf, evs)` after the struct literal, before `ContainerCreate`.
- [environment/docker/stats.go](https://github.com/PerrysSpace/GWings/blob/main/environment/docker/stats.go) — same idea, keep upstreams struct fields, re-add `GPU: e.gpuStats(),`.
- [config/config_docker.go](https://github.com/PerrysSpace/GWings/blob/main/config/config_docker.go) — a conflict here means upstream touched `DockerConfiguration`. The `Devices` field is purely additive, keep both sides.
- [README.md](https://github.com/PerrysSpace/GWings/blob/main/README.md) — resolve however makes sense; it's an append, not a replacement, so conflicts should be rare and trivial.

Then: 
```bash
go build ./...
go vet ./config/... ./environment/...
go test ./environment/... ./config/... -count=1
git rebase --continue
```
Only once that passes:
```bash
git branch -f main "rebase-onto-$TARGET"
git checkout main
git push --force-with-lease origin main
```
`main` is force-pushed only for this rebase — nowhere else.
