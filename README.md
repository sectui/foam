```
█▀▀ █▀▀ █▀▀ ▀█▀ █ █ █   ▄▀ █▀▀ █▀█ ▄▀█ █▀▄▀█
▄▄█ ██▄ █▄▄  █  █▄█ █ ▄▀   █▀  █▄█ █▀█ █ ▀ █
```

# Foam

Reusable components for [Bubble Tea v2](https://charm.land/bubbletea) apps of
the [foam](https://github.com/anotherhadi/foam) family.

## Install

```sh
go get github.com/anotherhadi/foam@latest
```

Versions follow the `v*` tags, with their changelog on the
[GitHub releases](https://github.com/anotherhadi/foam/releases).

## Configuration

To change the defaults, put a `~/.config/foam/config.yaml` file: it is shared
by all the foam tools. You only need the keys you want to change, the rest
falls back to the defaults.

```yaml
borderStyle: heavy
nerdFonts: true
```

The default config, with every key and its comments, is in
[`defaults.yaml`](defaults.yaml). Run a tool with `--print-config` to see the
merged config it ends up with.
