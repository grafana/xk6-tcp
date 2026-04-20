const fail = require("k6/execution").test.fail
const assert = (cond, msg) => { if (!cond) { fail(msg) } }
const tcp = require("k6/x/tcp")

exports.default = async () => {
  // 1. write() returns true on success
  {
    const socket = new tcp.Socket()
    let returnVal
    socket.on("connect", () => { returnVal = socket.write("Hello") })
    const closed = new Promise((resolve) => { socket.on("close", resolve) })
    socket.on("data", () => { socket.destroy() })
    socket.on("error", (err) => { fail(`unexpected error: ${err}`) })
    socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
    await closed
    assert(returnVal === true, `write() should return true on success, got ${returnVal}`)
  }

  // 2a. write() returns false on prepare-time error (bad encoding) handled via error event
  {
    const socket = new tcp.Socket()
    let returnVal
    const errored = new Promise((resolve) => { socket.on("error", () => resolve()) })
    socket.on("connect", () => { returnVal = socket.write("Hello", { encoding: "latin1" }) })
    socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
    await errored
    socket.destroy()
    assert(returnVal === false, `write() should return false on prepare-time error, got ${returnVal}`)
  }

  // 2b. write() returns false on execute-time error (no connection) handled via error event
  {
    const socket = new tcp.Socket()
    let returnVal
    const errored = new Promise((resolve) => { socket.on("error", () => resolve()) })
    returnVal = socket.write("Hello")  // conn is nil → writeExecute fails
    await errored
    socket.destroy()
    assert(returnVal === false, `write() should return false on execute-time error, got ${returnVal}`)
  }

  // 3. on() returns this for chaining
  {
    const socket = new tcp.Socket()
    let chainWorks = false
    const closed = new Promise((resolve) => { socket.on("close", resolve) })
    const result = socket
      .on("connect", () => { socket.destroy() })
      .on("error", (err) => { fail(`unexpected error: ${err}`) })
    chainWorks = result === socket
    socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
    await closed
    assert(chainWorks, "on() should return this for chaining")
  }

  // 4a. destroy(error) fires error event and preserves the Error object
  {
    const socket = new tcp.Socket()
    let errorFired = false
    let receivedError
    const errored = new Promise((resolve) => {
      socket.on("error", (err) => { errorFired = true; receivedError = err; resolve() })
    })
    const closed = new Promise((resolve) => { socket.on("close", resolve) })
    socket.on("connect", () => { socket.destroy(new Error("test reason")) })
    socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
    await errored
    await closed
    assert(errorFired, "destroy(error) should fire the error event")
    assert(receivedError instanceof Error, "destroy(error) should pass an Error to the error handler")
    assert(receivedError.message === "test reason",
      `destroy(error) should preserve error message, got ${receivedError && receivedError.message}`)
  }

  // 4b. port as string (env var pattern)
  {
    const socket = new tcp.Socket()
    const closed = new Promise((resolve) => { socket.on("close", resolve) })
    socket.on("connect", () => { socket.destroy() })
    socket.on("error", (err) => { fail(`unexpected error: ${err}`) })
    socket.connect(String(__ENV.TCP_ECHO_PORT), __ENV.TCP_ECHO_HOST)
    await closed
  }
}
