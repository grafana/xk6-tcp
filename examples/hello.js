import { Socket } from "k6/x/tcp";

/**
 * Basic TCP socket example demonstrating async-first flow.
 * This example connects to a TCP server, sends a message, and receives a response.
 */
export default async function () {
    const socket = new Socket();
    const closed = new Promise((resolve) => {
        socket.on("close", () => {
            console.log("Connection closed");
            resolve();
        });
    });

    socket.on("data", (data) => {
        const response = String.fromCharCode.apply(null, new Uint8Array(data));
        console.log("Received:", response);
        socket.destroy();
    });

    socket.on("error", (err) => {
        console.error("Socket error:", err);
    });

    // Connect to echo server (use with-echo wrapper to start server automatically)
    const host = __ENV.TCP_ECHO_HOST || "localhost";
    const port = __ENV.TCP_ECHO_PORT || "8080";

    await socket.connectAsync(port, host);
    console.log("Connected to TCP server");
    await socket.writeAsync("Hello, TCP!");
    await closed;
}
