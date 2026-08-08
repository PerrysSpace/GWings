# Pelican Wings

Wings is Pelican's server control plane, built for the rapidly changing gaming industry and designed to be
highly performant and secure. Wings provides an HTTP API allowing you to interface directly with running server
instances, fetch server logs, generate backups, and control all aspects of the server lifecycle.

In addition, Wings ships with a built-in SFTP server allowing your system to remain free of Pelican specific
dependencies, and allowing users to authenticate with the same credentials they would normally use to access the Panel.

## Documentation

* [Panel Documentation](https://pelican.dev/docs/panel/getting-started)
* [Wings Documentation](https://pelican.dev/docs/wings/install)
* Or, get additional help [via Discord](https://discord.gg/pelican-panel)

## Host device passthrough

This fork adds optional host device passthrough to server containers (e.g.
`/dev/dri` for GPU access), gated by an allowlist the node administrator
defines in `config.yml`. A server can only opt into a group that is defined
there — it can never supply or influence a device path itself.

### Configuring an allowlist

```yaml
docker:
  devices:
    gpu:
      paths: ["/dev/dri"]
      groups: ["44", "104"]
```

`paths` are host device paths mapped 1:1 into the container. `groups` are
supplemental Linux groups the container process is added to — **use numeric
GIDs, not names**. Docker resolves `GroupAdd` names against the *container
image's* `/etc/group`, not the host's, so a name like `video` silently grants
nothing if the image doesn't define that group. Find the host GIDs with:

```bash
getent group video render
```

### Opting in from a server (egg)

A server opts into a group named gpu by setting the environment variable
`ENABLE_GPU=1` (via its egg). Any other value, or the variable being unset,
means the group stays off. This variable can only ever turn a predefined
group on or off — it is never interpreted as a path.

Restarting the server (not reinstalling) is enough to apply a change, since
Wings recreates the container on every start.

## Reporting Issues

Feel free to report any wings specific issues or feature requests in [GitHub Issues](https://github.com/pelican-dev/wings/issues/new).
