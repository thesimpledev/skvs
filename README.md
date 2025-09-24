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

## Binary Protocol (Planned, Compact)

The protocol is designed to be tiny and efficient. Each message is fixed size, encrypted client-side and decrypted server-side.  
Our design goal is to keep messages well under 10 MB so they always fit cleanly in a single TCP packet stream.

---

### Layout

| Offset | Size   | Field   | Notes                                               |
| ------ | ------ | ------- | --------------------------------------------------- |
| 0      | 1 B    | Command | 0=SET, 1=GET, 2=DELETE, 3=EXISTS (up to 256 total). |
| 1      | 4 B    | Flags   | 32-bit bitmask. Each bit is an independent flag.    |
| 5      | 128 B  | Key     | ASCII/UTF-8 string, null-padded if shorter.         |
| 133    | 892 B  | Value   | ASCII/UTF-8 string, null-padded if shorter.         |
| 1025   | 64 B   | API Key | Fixed 64-char ASCII secret, validated post-decrypt. |
| Total  | 1089 B | Message | Fixed size, fully encrypted.                        |

---

### Flags Bitmask

| Bit  | Meaning   | Notes                                     |
| ---- | --------- | ----------------------------------------- |
| 0    | Overwrite | Allow overwriting existing values         |
| 1    | Old       | Return the previous value (even if empty) |
| 2    | Reserved  |                                           |
| 3    | Reserved  |                                           |
| 4    | Reserved  |                                           |
| 5    | Reserved  |                                           |
| 6    | Reserved  |                                           |
| 7–31 | Reserved  | Full 32-bit space allows future expansion |

---

### Example

- **SET key=foo, value=bar**
  - Command = `0` (SET)
  - Flags = `00000001` (overwrite=true, old=false)
  - Key field = `"foo"` + 125 null bytes
  - Value field = `"bar"` + 889 null bytes
  - API Key = 64 ASCII characters (configured on server, required to process)

---

### Benefits

- Compact: only **1089 bytes** per message.
- **1B command** = 256 possible command codes → enough for a small, purpose-built KV protocol.
- **4B flags** = 32 independent toggles for features.
- Parsing is O(1) with fixed offsets.
- Fully encrypted for confidentiality and integrity.
- Lightweight memory use: ~1.1 KB per entry (≈1.1 GB for 1M entries before Go map overhead).

---

### Notes

- The entire 1089-byte message is encrypted/decrypted in one operation (AES-256 GCM recommended).
- Keys and values are treated as ASCII/UTF-8.
- Multi-byte UTF-8 characters reduce effective count (e.g., emoji).
- For JWT expiry (primary use case), keys are ASCII (UUIDs, hashes) and values are timestamps, so no limitation.
