# Streaming brotli over SSE

A toy server and client that stream the files of a directory as
brotli-compressed Server-Sent Events, in three compression lifecycles.

> Demo, not production code: no dictionary negotiation, auth, or backpressure.

## Modes

The client picks a mode per request via `?mode=`; one running server serves all
three. Each event is length-prefixed and base64-encoded into an SSE `data:` line.

| mode     | lifecycle                                                            | per-event independent? |
| -------- | ------------------------------------------------------------------- | ---------------------- |
| `reset`  | one self-contained brotli stream per event; `Reset` between events  | yes                    |
| `dict`   | like `reset`, but every writer shares a compound dictionary         | yes                    |
| `stream` | one long-lived stream; never `Reset`, `Flush` after each event      | no (continuous)        |

`stream` keeps the compression window across events, so each event can
back-reference everything before it. `Flush` ends a block without finalizing the
stream, so each event's bytes are delivered immediately while the shared context
lives on.

## Run

```sh
go run ./server <port> <dir> <dict-file>
go run ./client [-dict <dict-file>] [-log <csv>] '<url>?mode={reset,dict,stream}'
```

`-dict` is required for `dict` mode (the decoder needs the same dictionary).
`-log` appends `mode,frame,compressed_bytes` per frame. The server also accepts
`&level=` and `&delay=` query params.
