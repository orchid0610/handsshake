# HandSSHake

A clean, fast SSH client for the desktop. Your hosts and credentials live in
an encrypted local vault, and nothing leaves your machine.

## Features

- Tabbed SSH terminals (xterm.js)
- Encrypted vault (Argon2id + XChaCha20-Poly1305) with a recovery key
- Password and private-key authentication, including passphrase-protected keys
- Host key verification with a trust-on-first-use prompt
- Search hosts with Ctrl+K

## Development

Requires Go, Node.js, the Wails v2 CLI, and (on Ubuntu) `libwebkit2gtk-4.1-dev`.

    make dev      # run with live reload
    make test     # run the Go tests
    make build    # build a release binary into build/bin