const fail = require("k6/execution").test.fail
const assert = (condition, message) => { if (!condition) { fail(message) } }

const tcp = require("k6/x/tcp")

exports.default = async () => {
  {
    const socket = new tcp.Socket()
    let connectFired = false
    let dataBeforeConnect = false

    const closed = new Promise((resolve) => {
      socket.on("close", resolve)
    })

    socket.on("connect", () => {
      connectFired = true
    })
    socket.on("data", () => {
      if (!connectFired) {
        dataBeforeConnect = true
      }

      socket.destroy()
    })
    socket.on("error", (err) => {
      fail(`unexpected error in connect/data ordering case: ${err}`)
    })

    socket.connect(__ENV.TCP_BANNER_PORT, __ENV.TCP_ECHO_HOST)
    await closed

    assert(connectFired, "connect should fire in banner server case")
    assert(!dataBeforeConnect, "data must not fire before connect")
  }

  {
    const socket = new tcp.Socket()
    let errorFired = false
    let closeBeforeError = false

    const closed = new Promise((resolve) => {
      socket.on("close", () => {
        if (!errorFired) {
          closeBeforeError = true
        }

        resolve()
      })
    })

    socket.on("error", () => {
      errorFired = true
    })
    socket.on("connect", () => {
      socket.destroy(new Error("ordered"))
    })

    socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
    await closed

    assert(errorFired, "error should fire when destroy(error) is used")
    assert(!closeBeforeError, "close must not fire before error when destroy(error) is used")
  }
}
