import tcp from "k6/x/tcp"

export default async function () {
  const socket = new tcp.Socket()

  socket.on("connect", () => {
    console.log("Connected")
    socket.write("Hey there\n");
  })

  socket.on("data", (data) => {
    console.log("Received data")
    const str = String.fromCharCode.apply(null, new Uint8Array(data))
    console.log(str)
    socket.destroy()
  })

  const prom = new Promise((resolve) => {
    socket.on("close", () => {
      console.log("Connection closed")
      resolve()
    })
  })

  socket.on("error", (err) => {
    console.log(`Socket error: ${err}`);
  })

  prom.then(() => {
    console.log("data handler was not called")
  })

  socket.connect(__ENV.TCP_ECHO_PORT, __ENV.TCP_ECHO_HOST)
  console.log("after connect")

  await prom
}
