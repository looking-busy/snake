# snake

A snake game that looks like a Spring Boot log tail. Needs a terminal at least 119x16; it refuses
to start, or exits, if the window is smaller than that.

```
make build && ./snake            # default 150ms per step
./snake -t 80ms                  # faster; --tick works too
```

The snake waits for your first key. Steer with arrows, `wasd`, or `hjkl`. Space pauses and hides the board behind log traffic.
`q`, `Esc`, or `Ctrl-C` shuts the "server" down.
Hitting a wall or yourself throws an exception and the program exits with status 1; your score
is the request latency on the `Completed 200 OK (N ms)` line just below the stack trace.
