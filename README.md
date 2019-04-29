# antsshd

SSH daemon with authentication delegated to a remote endpoint.

## Configuration

`antsshd` load config file `config.yml` from a directory, default to `/etc/antsshd`

See `testdata/config.yml` for detail

## Authentication Protocol

Every time a client try to connect `antsshd` (or create a new ssh channel), `antsshd` will ask remote endpoint for permission.

```text
POST http://example.com/antssh-endpoint

{
    "hostname"  : "app1.example.com"    // hostname, can be overridden in config.yml
    "user"      : "root",               // the linux user which client want to log in
    "public_key": "SHA256:sP1TWp04iqpM5h87qiVa5TtAWiCOlC95/FYiPe7M3hk",     // sha256 fingerprint of the client
    "type": "handshake",                // possible values are "handshake", "session", "direct-tcpip", "forward-tcpip", see below

    // type "handshake", the initial stage of ssh connection
    //
    // handshake has no extra parameters

    // type "session", when client execute command `ssh app1.example.com`
    //
    // session has no extra parameters

    // type "direct-tcpip", client want to create a tcp connection using 'antsshd' as a proxy
    //
    "target_host": "app2.example.com",  // target host of direct-tcpip
    "target_port": 80,                  //  target port of direct-tcpip

    // type "forward-tcpip", client want to listen a port on this server
    //
    "bind_host": "localhost",           // bind host of forward-tcpip
    "bind_port": 80,                    // bind port of forward-tcpip
}
```
