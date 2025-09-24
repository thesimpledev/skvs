# Simple Key Value Server

A tiny UDP key–value server for internal use where you own both client and server and want something lighter than Valkey/Redis. It’s intentionally minimal, fast, and easy to reason about.

- Transport: UDP (one datagram per request/response)
- Payload: compact fixed-size binary protocol
- Concurrency: per-request goroutine; in-memory map guarded by sync.RWMutex
- Persistence: none (in-memory only)
- Security: all payloads are AES-256-GCM encrypted (client-side encryption, server-side decryption).

---

## Run

- Port defaults to 4040 when PORT is unset or invalid.
- Example:

  PORT=4040 SKVS_ENCRYPTION_KEY=12345678901234567890123456789012 go run ./cmd/server

---

## Configuration

The server and client are configured via environment variables:

| Variable            | Description                                                   | Notes                          |
| ------------------- | ------------------------------------------------------------- | ------------------------------ |
| PORT                | UDP port for the server to bind. Defaults to 4040.            | Must be numeric.               |
| SKVS_ENCRYPTION_KEY | 32-byte key for AES-256-GCM encryption. Required.             | Must be exactly 32 bytes long. |
| SKVS_TIMEOUT        | Default timeout (in seconds) for client operations. Optional. | Defaults to 5.                 |

---

## Binary Protocol

Each message is a fixed-size 1029-byte frame.  
The entire frame is encrypted before transport. On the wire, the ciphertext size is 1029 + nonce (12) + tag (16) = 1057 bytes.

### Layout

| Offset | Size   | Field   | Notes                                               |
| ------ | ------ | ------- | --------------------------------------------------- |
| 0      | 1 B    | Command | 0=SET, 1=GET, 2=DELETE, 3=EXISTS (up to 256 total). |
| 1      | 4 B    | Flags   | 32-bit bitmask; each bit is an independent toggle.  |
| 5      | 128 B  | Key     | UTF-8 string, null-padded if shorter.               |
| 133    | 892 B  | Value   | UTF-8 string, null-padded if shorter.               |
| Total  | 1029 B | Frame   | Fixed size plaintext, encrypted as a whole.         |

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
- Plaintext frames are always 1029 bytes; ciphertext datagrams are 1057 bytes.
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

- Complete binary protocol encoder/decoder.
- Client library.
- End-to-end encryption of request and response payloads (AES-256-GCM).
- Key management via SKVS_ENCRYPTION_KEY env var (validate length).
- Switch encryption/decryption to log+drop on error, not fatal exit.
- Document 12-byte nonce size in protocol and README.
- Optimize buffer reuse with sync.Pool (server).
- Add debug logging for decoded commands.
- Evaluate moving key normalization to decode step.
- Document CLI flag usage order (--overwrite, --old before command).
- Client.Send: consider returning []byte instead of string.
- Client: use protocol.DefaultTimeout (5s) instead of hardcoded 2s.
- Client: support context.Context for cancellation and timeout.
