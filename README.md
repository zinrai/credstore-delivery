# credstore-delivery

A command-line tool to deliver secret values into the systemd credstore, `/etc/credstore.encrypted`, of remote hosts with [systemd-creds](https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html).

## Overview

`credstore-delivery push` runs on a deploy server. It reads a manifest of secret names, fetches each name once from a Unix socket, and passes each value over ssh, on stdin, to `credstore-delivery put` on every host given with `-hosts`.

`credstore-delivery put` runs on each host. It encrypts the value with the host key into `/etc/credstore.encrypted/<credential>`, only when the value differs from the one already there, so that a path unit restarts the service only on a real change.

Values never touch the disk of the deploy server and never appear in arguments. credstore-delivery does not know where the values come from: any program that serves the socket contract below can back it.

## Socket contract

HTTP over a Unix socket:

```
GET /<name>   200  body is the value, byte for byte
              404  no such name
              any other status is a failure
```

`<name>` is one or more segments of letters, digits, `_` and `-`, joined by `/`, such as `web/db-password`. The credential name on the host replaces `/` with `.`: `web.db-password`. Names contain no `.`, so two names never map to the same credential.

[zinrai/sakura-secrets-socket](https://github.com/zinrai/sakura-secrets-socket) serves this contract from SAKURA Cloud Secret Manager.

[`example/store.py`](./example/store.py) serves it from a dict in the script, on `./store.sock`, to try credstore-delivery without a secret store.

## Requirements

- Deploy server: `ssh` to every host, as the account given with `-user`, without a password prompt
- Hosts: systemd 250 or later, `credstore-delivery` on the `PATH` of `sudo`, and an account that may run `credstore-delivery put` through `sudo` without a password:

```
creds ALL=(root) NOPASSWD: /usr/local/bin/credstore-delivery put *
```

Give the account only the privilege `put` needs, and a key of its own: it can then replace any credential, but cannot get a root shell.

## Usage

The manifest lists what to deliver, one service's secret names; where to deliver them is given with `-hosts`, from whatever already lists the hosts:

```yaml
- web/db-password
- shared/api-token
```

```bash
$ credstore-delivery push -socket "$XDG_RUNTIME_DIR/secrets.sock" -manifest web.yaml -hosts web1,web2 -user creds
web1 [create] web/db-password
web1 [unchanged] shared/api-token
web2 [update] web/db-password
web2 [unchanged] shared/api-token
```

`push` fetches every value before it touches a host, and stops if the socket does not serve any one of them. It then runs, for each host and credential:

```bash
ssh -o BatchMode=yes -l <user> <host> sudo -n credstore-delivery put -name <credential>
```

It goes on past a failed host, lists the failed hosts at the end and exits 1. It does not retry; deliver to the failed hosts again by giving only them with `-hosts`.

The host key `/var/lib/systemd/credential.secret` is created by `systemd-creds` on the first `put`. A credential that no longer decrypts is an error; remove the file by hand and deliver it again. This happens to every credential of a host whose machine ID changed: systemd binds the host key to the machine ID and discards the key on the first use under a new one.

## Consuming a credential

A service takes a credential with a drop-in such as

```ini
[Service]
LoadCredentialEncrypted=web.db-password:/etc/credstore.encrypted/web.db-password
```

and reads it from `$CREDENTIALS_DIRECTORY/web.db-password`. To restart the service when `put` changes the value, watch the file with a path unit whose `Unit=` names a oneshot that runs `systemctl try-restart`; without `Unit=`, a path unit starts the service of its own name. `try-restart` does not start a stopped service, so deliver the credentials before starting a new service.

## License

This project is licensed under the [MIT License](./LICENSE).
