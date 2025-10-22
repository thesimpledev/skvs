# Simple Key Value Store

The Simple Key Value Store is a tiny UDP key–value server for a personal project where I needed something lighter than Valkey or Redis.. It’s intentionally minimal, fast, and easy to reason about.

[![Go Report Card](https://goreportcard.com/badge/github.com/thesimpledev/skvs)](https://goreportcard.com/report/github.com/thesimpledev/skvs)
[![License](https://img.shields.io/github/license/thesimpledev/skvs)](https://github.com/thesimpledev/skvs/blob/master/LICENSE)
[![Release](https://img.shields.io/github/v/release/thesimpledev/skvs)](https://github.com/thesimpledev/skvs/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/thesimpledev/skvs)](https://github.com/thesimpledev/skvs)
[![CI](https://github.com/thesimpledev/skvs/actions/workflows/ci.yml/badge.svg)](https://github.com/thesimpledev/skvs/actions)
[![codecov](https://codecov.io/gh/thesimpledev/skvs/branch/master/graph/badge.svg)](https://codecov.io/gh/thesimpledev/skvs)


## Features

- Transport: UDP (one datagram per request/response)
- Payload: compact fixed-size binary protocol
- Concurrency: per-request goroutine; in-memory map guarded by sync.RWMutex
- Persistence: none (in-memory only)
- Security: all payloads are AES-256-GCM encrypted (client-side encryption, server-side decryption).

### Commands

- `set <key> <value>` – store a value and returns the set value
- `get <key>` – retrieve a value - always returns a value even if it is empty
- `delete <key>` – remove a key - returns removed key
- `exists <key>` – check if a key exists - currently returns a string true/false

### Flags

- `--overwrite` allows existing key to be overwritten on set
- `--old` returns the previous key independent of any other flags

---

## Library Client

The Go client library is provided as a thin wrapper around the internal protocol.
It requires that every operation is called with a `context.Context` that has a deadline set.
If a deadline is not provided, the call will fail immediately.

### Example

```go
import (
    "context"
    "time"
    "github.com/thesimpledev/skvs/cmd/client_lib"
)

func main() {
    c, err := skvs.New("localhost:4040")
    if err != nil {
        panic(err)
    }
    defer c.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    _, err = c.Set(ctx, "foo", "bar", true, false)
    if err != nil {
        panic(err)
    }

    val, err := c.Get(ctx, "foo")
    if err != nil {
        panic(err)
    }
    fmt.Println("Value:", val)
}


## CLI Client Usage

A simple CLI is provided for local development and testing.

### Build and Run

Start the server in one terminal:

    PORT=4040 SKVS_ENCRYPTION_KEY=12345678901234567890123456789012 go run ./cmd/server

Then in another terminal run the CLI:

    go run ./cmd/client_cli  [--overwrite] [--old] <command> <key> [value]



### Examples

    go run ./cmd/client_cli set foo bar
    go run ./cmd/client_cli get foo
    go run ./cmd/client_cli --overwrite set foo baz
    go run ./cmd/client_cli --overwrite --old set foo qux
    go run ./cmd/client_cli delete foo
    go run ./cmd/client_cli exists foo

### Notes

- Flags (`--overwrite`, `--old`) must be provided **before** the command due to Gos stdlib `flag` package parsing rules.
- The CLI always applies the default timeout (`protocol.Timeout`) for requests.


---

## Configuration

The server and client are configured via environment variables:

| Variable            | Description                                        | Notes                          |
| ------------------- | -------------------------------------------------- | ------------------------------ |
| PORT                | UDP port for the server to bind. Defaults to 4040. | Must be numeric.               |
| SKVS_ENCRYPTION_KEY | 32-byte key for AES-256-GCM encryption. Required.  | Must be exactly 32 bytes long. |

---

## Binary Protocol

Each message is a fixed-size 1024-byte frame.
The entire frame is encrypted before transport. On the wire, the ciphertext size is 996 + nonce (12) + tag (16) = 1024 bytes.

### Layout

| Offset | Size   | Field   | Notes                                               |
| ------ | ------ | ------- | --------------------------------------------------- |
| 0      | 1 B    | Command | 0=SET, 1=GET, 2=DELETE, 3=EXISTS (up to 256 total). |
| 1      | 4 B    | Flags   | 32-bit bitmask; each bit is an independent toggle.  |
| 5      | 128 B  | Key     | UTF-8 string, null-padded if shorter.               |
| 133    | 863 B  | Value   | UTF-8 string, null-padded if shorter.               |
| Total  | 996 B | Frame   | Fixed size plaintext, encrypted as a whole.         |

---

### Command Table

| Code  | Command | Description                                     |
| ----- | ------- | ----------------------------------------------- |
| 0     | SET     | Store a value at a key, respecting flags.       |
| 1     | GET     | Retrieve the value at a key (empty if missing). |
| 2     | DELETE  | Remove the key, returning the old value.        |
| 3     | EXISTS  | Return "true" if key exists, "false" if not.    |
| 4–255 | —       | Reserved for future use.                        |

---

### Flags Bitmask

| Bit  | Meaning   | Notes                                      |
| ---- | --------- | ------------------------------------------ |
| 0    | Overwrite | Allow overwriting existing values.         |
| 1    | Old       | Return the previous value (even if empty). |
| 2–31 | Reserved  | Full 32-bit space allows future expansion. |

---

## Operational Notes

- One UDP datagram = one operation.
- Plaintext frames are always 1024 bytes; ciphertext datagrams are 1057 bytes.
- Server responses are short binary or string payloads. Errors are returned as generic "ERROR: failed to process message".
- Reads scale via RLock for GET/EXISTS; writes (SET/DELETE) take a short exclusive Lock.
- Data is volatile — lost on restart.
- No authentication/authorization — security is enforced by encryption only.

---

## Non-Goals

- Persistence, replication, clustering, TTLs, eviction policies.
- Complex data structures or scripting.
- Streaming or multi-message pipelines.

---

## Todo

- Add debug logging for decoded commands.
- Response typing: External client (Do) returns string, while the internal client returns []byte. May want to address


```
