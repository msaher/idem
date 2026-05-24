# `idem`

Package `idem` provides declarative system configuration primitives through an idempotent API. It's inspired by tools like ansible.

```go
h := idem.NewHost("127.0.0.1", sshConfig).WithPort(8022).WithSudo(true)

idem.Package("nginx").State("present").Run(h)
idem.User("nginx").Run(h)
idem.File("/var/www").State("directory").Owner("nginx").Run(h)
idem.Command("nginx").Run(h)
```

# Why?

Infrastructure as code tools are great, but they often use a configuration language instead of a programming language. `idem` is still a Go library, no YAML, no python, and no CLI.

# Architecture

`idem` works by pushing a statically linked Go binary into the server. The remote binary examines current state and makes the necessary changes to reach the desired state.

# Status

`idem` is usable, but keep in mind its still work in progress. Contributions welcome!
