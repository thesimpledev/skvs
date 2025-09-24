# Simple Key Value Server

A tiny TCP key–value server for internal use where you own both client and server and want something lighter than Valkey/Redis. It’s intentionally minimal, fast, and easy to reason about.

- **Transport:** TCP
- **Payload:** single JSON message per connection (≤ 1024 bytes)
- **Response:** plain string
- **Concurrency:** per-connection goroutine; in-memory map guarded by `sync.RWMutex`
- **Persistence:** none (in-memory only)

---

## Run

- Port defaults to 4000 when `PORT` is unset or invalid.
- Example: PORT=4000 go run .

---

## Protocol

Each client connection sends exactly one JSON object and receives one string response, then the server closes the connection.

JSON fields (all strings except flags):

- `command` (int): 0=SET, 1=GET, 2=DELETE, 3=EXISTS
- `key` (string)
- `value` (string, for SET)
- `overwrite` (bool, optional)
- `old` (bool, optional)

Example payload (as a single line):
{"command":0,"key":"foo","value":"bar","overwrite":false,"old":false}

On malformed/unsupported input the server replies:
ERROR: failed to process message

---

## Commands

### SET (command=0)

Stores `value` at `key`.

Options (independent):

- `overwrite`: if true, replace existing values; if false and key exists, do not modify.
- `old`: if true, the response is the previous value (empty string if none); if false, the response is the resulting value when an insert/overwrite occurs, or the existing value if no change happened.

Behavior summary:

- New key, `overwrite` ignored:
  - `old=false` → returns new value
  - `old=true` → returns empty string
- Existing key, `overwrite=false`:
  - Value unchanged; returns existing value (regardless of `old`)
- Existing key, `overwrite=true`:
  - Value replaced; `old=true` returns old value, `old=false` returns new value

### GET (command=1)

Returns the stored value. If the key is missing, returns an empty string.

### DELETE (command=2)

Deletes the key and returns the previous value (empty string if it wasn’t set).

### EXISTS (command=3)

Returns the string "true" if the key exists, otherwise "false".

---

## Operational Notes

- **Message size:** up to 1024 bytes per request. Keep payloads small; this server is optimized for short, single-operation messages.
- **Concurrency:** read-heavy loads scale via `RLock` for `GET`/`EXISTS` and exclusive `Lock` for `SET`/`DELETE`.
- **Logging:** server logs with `slog`; client responses never include internal error details.
- **Data lifetime:** contents are lost on process exit/restart. No persistence, TTL, or replication.

---

## Non-Goals

- Authentication/authorization
- Persistence or durability guarantees
- Complex types or binary payloads
- Multi-message streaming per connection

---

## Todo

Implement encryption for payloads and responses.
Create client library

## Binary Protocol (Planned)

To reduce parsing overhead and keep message sizes consistent, the server will support a fixed-size binary protocol. Each message is exactly **1024 bytes**.

### Layout

| Offset | Size   | Field   | Notes                                      |
| ------ | ------ | ------- | ------------------------------------------ |
| 0      | 1 byte | Flags   | bit 0 = overwrite, bit 1 = old             |
| 1      | 1 byte | Command | 0=SET, 1=GET, 2=DELETE, 3=EXISTS           |
| 2      | 128 B  | Key     | ASCII/UTF-8 string, null-padded if shorter |
| 130    | 892 B  | Value   | ASCII/UTF-8 string, null-padded if shorter |
| Total  | 1024 B | Message | Fixed size                                 |

### Example

- **SET key=foo, value=bar**
  - Flags = `00000001` (overwrite=true, old=false)
  - Command = `0` (SET)
  - Key field = `"foo"` + 125 null bytes
  - Value field = `"bar"` + 889 null bytes

### Benefits

- Constant 1024-byte messages (simple to read/write).
- Parsing requires no dynamic length checks.
- Perfectly aligned with existing buffer size.
- Supports up to 128-character keys and 892-character values.
- Memory footprint: ~1 KB per entry (≈1 GB for 1M entries before Go map overhead).

### Notes

- Keys and values are treated as ASCII/UTF-8.
- Multi-byte UTF-8 characters reduce effective character count (e.g., emoji).
- For JWT expiry (the primary use case), keys will be ASCII (UUIDs, hashes, usernames) and values will be timestamps, so this is not a limitation.
