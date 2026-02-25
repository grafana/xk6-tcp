const fail = require("k6/execution").test.fail
const assert = (condition, message) => { if (!condition) { fail(message) } }

const tcp = require("k6/x/tcp")

exports.default = () => {
    const socket = new tcp.Socket({})

    let timeoutHandlerCalled = false
    socket.on("timeout", () => {
        timeoutHandlerCalled = true
        // User must manually destroy after timeout
        socket.destroy()
    })

    socket.on("connect", () => {
        // Set 2-second timeout
        socket.setTimeout(2000)
        // Don't write anything - let it timeout
    })

    const prom = new Promise((resolve) => {
        socket.on("close", () => {
            resolve()
        })
    })

    socket.on("error", (err) => {
        console.log(`Socket error: ${err}`)
    })

    prom.then(() => {
        assert(timeoutHandlerCalled, "timeout handler was not called")
    })

    socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
}
