# Answering an agent

siltide serves its snapshot over the Model Context Protocol, so an agent can
ask what the fleet is doing without anyone writing a client for our API first.

Everything it exposes is a read. No tool changes a device, runs a command,
writes a file or opens a connection, which is why the rest of this page is
short: there is no consent flow to design, because there is nothing to
consent to.

## Connecting

Over stdio, for a client that spawns siltide itself:

```json
{
  "mcpServers": {
    "siltide": {
      "command": "siltide",
      "args": ["--mcp-stdio"]
    }
  }
}
```

There is no token here, and none is needed: the client already holds whatever
privileges it started siltide with.

Run by hand, `--mcp-stdio` looks like a program that did not start. It is a
server: it reads a request on stdin and answers on stdout, and says nothing
until something asks it a question. It prints a line to stderr saying so,
because stderr is not the protocol stream, and if you type something at it
that is not JSON it says that too, once. To see it answer, hand it a request:

```sh
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | siltide --mcp-stdio
```

Over HTTP, for a client that connects to something already running:

```sh
siltide --mcp-http 127.0.0.1:8765
```

That address has to be loopback, and siltide refuses to start on anything
else. When `token:` is set in the config, the endpoint wants the same bearer
token `--listen` does. Requests must also arrive with a loopback `Host`
header, so a page in a browser on the same machine cannot be pointed at it.

## The tools

| Tool | What it answers |
| --- | --- |
| `fleet_summary` | how many devices, how many busy, idle, allocated or down, the power, the memory, the worst health, the alerts. Start here |
| `list_devices` | one line per device, narrowed by a filter |
| `device_detail` | every metric for one device, with its health notes, links, topology and processes |
| `list_processes` | the processes holding devices, with pod, namespace and workload where Kubernetes is present |
| `device_history` | recent values of one metric, oldest first, out of the history on disk |
| `recent_events` | thermal warnings, ECC errors, Xids, devices appearing and disappearing, and rules from the config |
| `health_report` | every device below full health with the reasons, the alerts firing, and what is allocated but idle |

`list_devices` and `list_processes` take the same filter language the
interface does, which is the part worth knowing:

```
util>80                  devices above eighty percent
util<5 procs>0           idle, but something is holding them
!vendor:nvidia           everything that is not NVIDIA
ns:ml temp>=70           the ml namespace, running hot
mem_used>8G              more than eight gigabytes in use
```

## What it does not do

- It cannot reset, reboot, kill, cordon or drain anything. siltide has never
  been able to; the protocol does not change that.
- It reads the snapshot the collector already produced, so an answer is at
  most one refresh interval old, and `fleet_summary` says when it was taken.
- It does not reach across the fleet by itself. Remote nodes appear because
  they are in the config, the same as in the interface.
