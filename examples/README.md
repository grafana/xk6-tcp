# xk6-tcp Examples

This directory contains example scripts demonstrating various features of the xk6-tcp extension.

## Running Examples

All examples can be run using k6 with the xk6-tcp extension:

```bash
# Build k6 with xk6-tcp (if not already built)
make build

# Run examples with the built-in echo server wrapper
./with-echo ./k6 run examples/hello.js
```

The `with-echo` wrapper automatically:
- Starts TCP and HTTP echo servers on random localhost ports
- Sets `TCP_ECHO_HOST`, `TCP_ECHO_PORT`, `HTTP_ECHO_HOST`, `HTTP_ECHO_PORT`, and `HTTP_ECHO_URL` environment variables
- Runs your k6 script with these servers available
- Cleans up servers when the script completes

### Using External Servers

You can also run examples against custom TCP servers:

```bash
# With custom host/port
TCP_ECHO_HOST=example.com TCP_ECHO_PORT=9000 ./k6 run examples/hello.js
```

Or start a standalone echo server:

```bash
# Using netcat (Linux/Mac)
while true; do nc -l 8080 -c 'xargs -0 echo'; done

# Then run without with-echo wrapper
TCP_ECHO_HOST=localhost TCP_ECHO_PORT=8080 ./k6 run examples/hello.js
```

## Examples Overview

### [hello.js](hello.js)
**Basic async-first programming**

Demonstrates the fundamental pattern of TCP communication using `connectAsync()`, `writeAsync()`, and socket event handlers.

**Key concepts:**
- Async-first API
- Basic connect/write/read pattern
- String data handling

### [hello_async.js](hello_async.js)
**Async/await pattern**

Shows how to use async methods (`connectAsync`, `writeAsync`) for cleaner code flow with promises.

**Key concepts:**
- Async/await operations
- Promise-based control flow
- Synchronous-style code

### [echo.js](echo.js)
**Bidirectional communication**

Demonstrates sending multiple messages and receiving echo responses in sequence.

**Key concepts:**
- Multiple write operations
- State management across callbacks
- Sequential message handling

### [timeout.js](timeout.js)
**Timeout handling**

Shows how to set and handle read timeouts for detecting idle connections.

**Key concepts:**
- `setTimeout()` method
- Timeout event handling
- Inactivity detection

### [options.js](options.js)
**Socket options and metrics tags**

Demonstrates using tags for organizing and filtering metrics in k6 output.

**Key concepts:**
- Socket constructor options
- Per-connection tags
- Per-operation tags
- Metrics organization

### [binary.js](binary.js)
**Binary protocol data**

Shows how to send and receive binary data using ArrayBuffer and Uint8Array.

**Key concepts:**
- ArrayBuffer usage
- Binary protocol headers
- Byte manipulation
- String.fromCharCode for binary-to-text conversion

### [multiple.js](multiple.js)
**Concurrent connections**

Demonstrates managing multiple TCP connections simultaneously with Promise coordination.

**Key concepts:**
- Multiple sockets
- Promise.all() coordination
- Concurrent operations
- Per-connection tagging

### [state.js](state.js)
**Socket state inspection**

Shows how to check socket properties like connection state and byte counters.

**Key concepts:**
- `ready_state` property
- `connected` property
- `bytes_written` counter
- `bytes_read` counter
- State lifecycle

### [smoke.test.js](smoke.test.js)
**Load testing example**

Demonstrates k6 load testing with TCP sockets, including multiple VUs, thresholds, and checks.

**Key concepts:**
- k6 options (vus, duration, thresholds)
- check() assertions
- Multiple concurrent VUs
- Performance validation
- Load testing patterns

### [tls.js](tls.js)
**TLS/SSL secure connections**

Shows how to establish encrypted TCP connections using TLS/SSL with the async API.

**Key concepts:**
- TLS encryption
- Secure connections
- HTTPS-like protocols
- k6 TLS configuration
- Certificate handling

### [tls_async.js](tls_async.js)
**TLS with async/await**

Demonstrates secure TLS connections using async/await pattern for better flow control.

**Key concepts:**
- TLS + async operations
- Promise-based TLS handshake
- Secure data exchange
- Connection lifecycle with TLS

### [tls_smtp.js](tls_smtp.js)
**SMTP over TLS example**

Shows how to connect to secure mail servers using TLS (SMTPS).

**Key concepts:**
- Protocol-specific TLS usage
- SMTP with implicit TLS
- Real-world TLS application
- Protocol handshakes

### [basic.js](basic.js)
**Minimal example**

The simplest possible socket creation example.

## Building with-echo

The `with-echo` binary is built from source in [../tools/with-echo](../tools/with-echo). To rebuild:

```bash
cd tools/with-echo
go build -o ../../with-echo
```

## Configuration

All examples support these environment variables:

- `TCP_ECHO_HOST` - Target host (default: localhost, automatically set by with-echo)
- `TCP_ECHO_PORT` - Target port (default: 8080, automatically set by with-echo)
- `HTTP_ECHO_HOST` - HTTP echo server host (automatically set by with-echo)
- `HTTP_ECHO_PORT` - HTTP echo server port (automatically set by with-echo)
- `HTTP_ECHO_URL` - Full HTTP echo server URL (automatically set by with-echo)

## Testing with k6 Options

You can combine examples with k6 load testing features:

```bash
# Run with multiple VUs
./k6 run --vus 10 --duration 30s examples/hello.js

# With custom thresholds
./k6 run --vus 5 --duration 10s \
  --threshold 'tcp_socket_duration<100' \
  examples/hello.js
```

## Common Patterns

### Event Handler Pattern
```javascript
socket.on("event", (data) => {
  // Handle event
});
```

### Async Pattern
```javascript
await socket.connectAsync(port, host);
await socket.writeAsync(data);
```

### Promise Coordination
```javascript
const promise = new Promise((resolve) => {
  socket.on("close", resolve);
});
await promise;
```

## Metrics

All examples generate k6 metrics:

- `tcp_socket_connecting` - Connection establishment time
- `tcp_socket_resolving` - DNS resolution time
- `tcp_socket_duration` - Total connection duration
- `tcp_sockets` - Number of sockets created
- `tcp_reads` - Number of read operations
- `tcp_writes` - Number of write operations
- `tcp_errors` - Number of errors
- `tcp_timeouts` - Number of timeouts
- `tcp_partial_writes` - Number of partial writes

## Troubleshooting

**Connection refused:**
- Make sure the TCP server is running
- Check firewall settings
- Verify host/port are correct

**Timeout:**
- Increase timeout value with `socket.setTimeout()`
- Check network connectivity
- Verify server is responding

**No data received:**
- Ensure server echoes data back
- Check write operation completed
- Verify data format matches server expectations

## See Also

- [Test files](../test/) - Additional examples used for testing
- [README](../README.md) - Main documentation
- [TypeScript definitions](../index.d.ts) - Complete API reference
