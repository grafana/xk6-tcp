import { Socket } from "k6/x/tcp";

/**
 * Basic TCP socket example demonstrating event-driven programming.
 * This example connects to a TCP server, sends a message, and receives a response.
 */
export default function () {
    const socket = new Socket();

    socket.on("connect", () => {
        console.log("Connected to TCP server");
        socket.write("Hello, TCP!");
    });

    socket.on("data", (data) => {
        const response = String.fromCharCode.apply(null, new Uint8Array(data));
        console.log("Received:", response);
        socket.destroy();
    });

    socket.on("close", () => {
        console.log("Connection closed");
    });

    socket.on("error", (err) => {
        console.error("Socket error:", err);
    });

    // Connect to echo server (use with-echo wrapper to start server automatically)
    const host = __ENV.TCP_ECHO_HOST || "localhost";
    const port = __ENV.TCP_ECHO_PORT || "8080";
    socket.connect(port, host);
}
