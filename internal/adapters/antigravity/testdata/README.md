# Antigravity fixtures

`usage.json` is synthetic. Release 1.1.11 documents `agy -p "/usage" --output-format json`
and that the command does not spend quota, but it does not publish a sample object.
The envelope (`status`, `command.name`, `command.data.groups[].buckets[].remaining_fraction`,
`reset_time`) is reconstructed from independent parsers of that command:

- https://docs.rs/usage-monitor-cli/latest/src/usage_monitor_cli/provider/antigravity.rs.html
- https://cdn.jsdelivr.net/npm/@tian.zuo/pi-antigravity@0.11.3/lib/usage.ts

This is not a live capture. A real `agy` payload is still required to confirm the parser.
