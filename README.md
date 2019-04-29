# antsshd

SSH daemon with authentication delegated to a remote endpoint.

## Configuration

`antsshd` load config file `config.yml` from a directory, default to `/etc/antsshd`

See `testdata/config.yml` for detail

## AntSSH Authentication Protocol

Every time a client try to connect `antsshd` (or create a new ssh channel), `antsshd` will ask remote endpoint for permission.

The request `antssh` will send:

```text
POST http://example.com/antssh-endpoint

{
    "hostname"  : "app1.example.com"    // hostname, can be overridden in config.yml
    "user"      : "root",               // linux user to login
    "public_key": "SHA256:sP1TWp04iqpM5h87qiVa5TtAWiCOlC95/FYiPe7M3hk",     // fingerprint of the client public key
    "type"      : "connect",            // possible values are "connect", "execute", "proxy", "forward", see below

    // "connect", initialize a ssh connection
    //
    // "connect" has no extra parameters

    // "execute", client want to start a terminal, execute a command or invoke a subsystem
    //
    // "execute" has no extra parameters

    // "proxy", client want to proxy a tcp connection (with 'ssh -L' command)
    //
    "proxy": {
        "host": "target.example.com",
        "port": 80
    },

    // "forward", client want to forward a port on server (with 'ssh -R' command)
    //
    "forward": {
        "host": "0.0.0.0",
        "port": 80
    }
}
```

The response an endpoint should reply:

```text
* HTTP 200, endpoint granted, `antsshd` will proceed the action

Plain Text: ok

* HTTP 400, endpoint denied, `antsshd` will refuse to proceed the action

Plain Text: error message

* HTTP 500, unexpected error occurred, `antsshd` will refuse to proceed the action

Anything

```

## Credits

Yanke Guo, MIT License
