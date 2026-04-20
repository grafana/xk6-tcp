const fail = require("k6/execution").test.fail
const assert = (cond, msg) => { if (!cond) { fail(msg) } }
const tcp = require("k6/x/tcp")

exports.default = async () => {
  // Case 1: prepare failure — pass a boolean, which hits the default branch in
  // connectPrepare() and returns errInvalidType. Strings are accepted (ToInteger()),
  // so "not-a-port" would NOT trigger this path.
  {
    const socket = new tcp.Socket()
    let rejected = false
    try {
      await socket.connectAsync(true)
    } catch (_) {
      rejected = true
    }
    socket.destroy()
    assert(rejected, "connectAsync should reject on invalid argument type")
  }

  // Case 2: execute failure — port 0 is reserved; connect() to it is always refused
  // on Linux regardless of what else is running on the machine.
  {
    const socket = new tcp.Socket()
    let rejected = false
    try {
      await socket.connectAsync(0, "127.0.0.1")
    } catch (_) {
      rejected = true
    }
    socket.destroy()
    assert(rejected, "connectAsync should reject on connection refused")
  }

  // Case 3: execute failure with error handler registered — promise must STILL reject.
  // The error handler is registered so handleError() takes the soft-error path
  // (returns nil), which is the path the old code used to silently resolve the promise.
  {
    const socket = new tcp.Socket()
    socket.on("error", () => {})
    let rejected = false
    try {
      await socket.connectAsync(0, "127.0.0.1")
    } catch (_) {
      rejected = true
    }
    socket.destroy()
    assert(rejected, "connectAsync should reject even when error handler is registered")
  }

  // Case 4: prepare failure with error handler registered — same soft-error path but
  // triggered at prepare time rather than execute time.
  {
    const socket = new tcp.Socket()
    socket.on("error", () => {})
    let rejected = false
    try {
      await socket.connectAsync(true)
    } catch (_) {
      rejected = true
    }
    socket.destroy()
    assert(rejected, "connectAsync should reject on prepare failure even when error handler is registered")
  }
}
