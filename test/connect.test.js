const fail = require("k6/execution").test.fail
const assert = (condition, message) => { if (!condition) { fail(message) } }

const tcp = require("k6/x/tcp")

exports.default = () => {
  const socket = new tcp.Socket({})

  let connectHandlerCalled = false
  socket.on("connect", () => {
    assert(socket.connected, "socket should be connected after connect event")
    assert(socket.ready_state === "open", `ready_state mismatch, expected 'open', got '${socket.ready_state}'`)

    connectHandlerCalled = true

    socket.destroy()
  })

  let closeHandlerCalled = false
  const prom = new Promise((resolve) => {
    socket.on("close", () => {
      closeHandlerCalled = true
      resolve()
    })
  })

  prom.then(() => {
    assert(connectHandlerCalled, "connect handler was not called")
    assert(closeHandlerCalled, "close handler was not called")
  })

  assert(!socket.connected, "socket should not be connected initially")

  socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)

  assert(socket.connected, "socket should be connected after connect call")
}
