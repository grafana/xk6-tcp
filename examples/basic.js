import { Socket } from "k6/x/tcp";

/**
 * Minimal TCP socket example.
 * This is the simplest possible usage - create a socket, connect, and close.
 */
export default function () {
  const socket = new Socket();

  socket.on("connect", () => {
    console.log("Connected!");
    socket.destroy();
  });

  socket.on("close", () => {
    console.log("Closed!");
  });

  socket.on("error", (err) => {
    console.error("Error:", err);
  });

  const host = __ENV.TCP_ECHO_HOST || "localhost";
  const port = __ENV.TCP_ECHO_PORT || "8080";
  socket.connect(port, host);
}
