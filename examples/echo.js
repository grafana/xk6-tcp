import { Socket } from "k6/x/tcp";

/**
 * Echo server example demonstrating bidirectional communication.
 * Sends multiple messages and verifies echo responses.
 */
export default function () {
    const socket = new Socket();
    const messages = ["Message 1", "Message 2", "Message 3"];
    let messageIndex = 0;
    let receivedCount = 0;

    socket.on("connect", () => {
        console.log("Connected to echo server");
        // Send first message
        socket.write(messages[messageIndex++]);
    });

    socket.on("data", (data) => {
        const response = String.fromCharCode.apply(null, new Uint8Array(data));
        console.log(`Echo response ${++receivedCount}:`, response);

        // Send next message or close
        if (messageIndex < messages.length) {
            socket.write(messages[messageIndex++]);
        } else {
            console.log("All messages sent and echoed");
            socket.destroy();
        }
    });

    socket.on("close", () => {
        console.log(`Connection closed. Received ${receivedCount} responses.`);
    });

    socket.on("error", (err) => {
        console.error("Socket error:", err);
    });

    const host = __ENV.TCP_ECHO_HOST || "localhost";
    const port = __ENV.TCP_ECHO_PORT || "8080";
    socket.connect(port, host);
}
