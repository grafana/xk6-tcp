import { Socket } from "k6/x/tcp";

/**
 * Timeout handling example.
 * Demonstrates setting and handling read timeouts for idle connections.
 */
export default function () {
    const socket = new Socket();

    socket.on("connect", () => {
        console.log("Connected to server");

        // Set a 5-second timeout for inactivity
        socket.setTimeout(5000);
        console.log("Timeout set to 5 seconds");

        // Optionally send a message
        // socket.write("Hello");
    });

    socket.on("data", (data) => {
        const response = String.fromCharCode.apply(null, new Uint8Array(data));
        console.log("Received data:", response);

        // Reset timeout after receiving data
        socket.setTimeout(5000);
    });

    socket.on("timeout", () => {
        console.log("Connection timeout - no data received for 5 seconds");
        // You must manually close after timeout
        socket.destroy();
    });

    socket.on("close", () => {
        console.log("Connection closed");
    });

    socket.on("error", (err) => {
        console.error("Socket error:", err);
    });

    const host = __ENV.TCP_ECHO_HOST || "localhost";
    const port = __ENV.TCP_ECHO_PORT || "8080";
    socket.connect(port, host);
}
