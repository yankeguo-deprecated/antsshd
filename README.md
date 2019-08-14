# antsshd

SSH daemon with authentication delegated to a remote endpoint.

## Configuration

`antsshd` load config file `config.yml` from a directory, default to `/etc/antsshd`

See `testdata/config.yml` for detail

## AntSSH Authentication Protocol

`antsshd` delegate all authentication to a remote gRPC endpoint, check https://github.com/antssh/types for details

## Credits

Guo Y.K., MIT License
