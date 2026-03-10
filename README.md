# epoch - epoch ↔ RFC3339

### Usage
`epoch <value>`

### Rules
  - If <value> is a pure number (optional leading +/-), it's treated as epoch.
    The tool will show interpretations for seconds, milliseconds, and nanoseconds.
  - If <value> contains ":" it's treated as RFC3339-like (with or without timezone).
  - Otherwise, the program fails.

### Outputs
  - For epoch input:
      - RFC3339 (UTC and Local) for each possible unit (s, ms, ns).
  - For RFC input:
      - Epoch (UTC) in s, ms, and ns.

### Examples
- epoch 1700000000
- epoch 2023-11-14T22:13:20Z

