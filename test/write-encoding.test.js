const fail = require("k6/execution").test.fail
const assert = (cond, msg) => { if (!cond) { fail(msg) } }
const tcp = require("k6/x/tcp")

const bytesToStr = (data) => String.fromCharCode.apply(null, new Uint8Array(data))

function writeEncodingTest(encoded, opts, expected) {
  const socket = new tcp.Socket()

  socket.on("connect", () => { socket.write(encoded, opts) })

  let received = false
  socket.on("data", (data) => {
    received = true
    assert(bytesToStr(data) === expected,
      `encoding ${opts.encoding}: expected '${expected}', got '${bytesToStr(data)}'`)
    socket.destroy()
  })

  const closed = new Promise((resolve) => { socket.on("close", () => resolve()) })

  socket.on("error", (err) => { fail(`socket error: ${err}`) })

  socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)

  return closed.then(() => {
    assert(received, `encoding ${opts.encoding}: data handler was not called`)
  })
}

function writeErrorTest(encoded, opts) {
  const socket = new tcp.Socket()
  const errored = new Promise((resolve) => {
    socket.on("error", () => resolve())
  })
  socket.on("connect", () => { socket.write(encoded, opts) })
  socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
  return errored.then(() => { socket.destroy() })
}

exports.default = async () => {
  // utf8 (default — no encoding specified)
  await writeEncodingTest("Hello", {}, "Hello")

  // explicit utf8
  await writeEncodingTest("Hello", { encoding: "utf8" }, "Hello")

  // base64: base64("Hello") = "SGVsbG8="
  await writeEncodingTest("SGVsbG8=", { encoding: "base64" }, "Hello")

  // hex: hex("Hello") = "48656c6c6f"
  await writeEncodingTest("48656c6c6f", { encoding: "hex" }, "Hello")

  // ascii — alias for raw byte conversion
  await writeEncodingTest("Hello", { encoding: "ascii" }, "Hello")

  // ArrayBuffer with non-default encoding — encoding must be ignored, raw bytes echoed
  await (() => {
    const socket = new tcp.Socket()
    const buf = new Uint8Array([72, 101, 108, 108, 111]).buffer  // "Hello"
    socket.on("connect", () => { socket.write(buf, { encoding: "hex" }) })
    let received = false
    socket.on("data", (data) => {
      received = true
      assert(bytesToStr(data) === "Hello",
        `ArrayBuffer with encoding: expected 'Hello', got '${bytesToStr(data)}'`)
      socket.destroy()
    })
    const closed = new Promise((resolve) => { socket.on("close", () => resolve()) })
    socket.on("error", (err) => { fail(`socket error: ${err}`) })
    socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
    return closed.then(() => {
      assert(received, "ArrayBuffer with encoding: data handler was not called")
    })
  })()

  // unsupported encoding — must fire error event
  await writeErrorTest("Hello", { encoding: "latin1" })

  // malformed base64 — must fire error event
  await writeErrorTest("not!!base64", { encoding: "base64" })

  // malformed hex — must fire error event
  await writeErrorTest("zzzz", { encoding: "hex" })

  // writeAsync execute-time failure — must reject AND fire error event
  // (no connection → writeExecute fails with errNoActiveConnection)
  // Wait for error event before destroying: reject() races ahead of the error-event
  // goroutine, so destroying immediately after await would cancel the loop first.
  {
    const socket = new tcp.Socket()
    let rejected = false
    let errorFired = false
    const errored = new Promise((resolve) => {
      socket.on("error", () => { errorFired = true; resolve() })
    })
    const closed = new Promise((resolve) => { socket.on("close", () => resolve()) })
    try {
      await socket.writeAsync("Hello")
    } catch (_) {
      rejected = true
    }
    await errored
    socket.destroy()
    await closed
    assert(rejected, "writeAsync should reject on execute-time failure")
    assert(errorFired, "writeAsync should fire error event on execute-time failure")
  }

  // writeAsync prepare failure — must reject AND fire error event
  {
    const socket = new tcp.Socket()
    let rejected = false
    let errorFired = false
    socket.on("error", () => { errorFired = true })
    const closed = new Promise((resolve) => { socket.on("close", () => resolve()) })
    await socket.connectAsync(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
    try {
      await socket.writeAsync("Hello", { encoding: "latin1" })
    } catch (_) {
      rejected = true
    }
    socket.destroy()
    await closed
    assert(rejected, "writeAsync should reject on unsupported encoding")
    assert(errorFired, "writeAsync should fire error event on unsupported encoding")
  }
}
