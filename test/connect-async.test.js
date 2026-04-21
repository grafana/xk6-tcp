const fail = require("k6/execution").test.fail
const assert = (condition, message) => { if (!condition) { fail(message) } }

const tcp = require("k6/x/tcp")

exports.default = async () => {
  const socket = new tcp.Socket({})

  let connectHandlerCalled = false
  const connected = new Promise((resolve) => {
    socket.on("connect", () => {
      assert(socket.connected, "socket should be connected after connect event")

      connectHandlerCalled = true
      resolve()
    })
  })

  let closeHandlerCalled = false
  const closed = new Promise((resolve) => {
    socket.on("close", () => {
      closeHandlerCalled = true
      resolve()
    })
  })

  assert(!socket.connected, "socket should not be connected initially")

  await socket.connectAsync(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)

  assert(socket.connected, "socket should be connected after connect call")
  await connected
  assert(connectHandlerCalled, "connect handler was not called")

  socket.destroy()
  await closed
  assert(closeHandlerCalled, "close handler was not called")
}
