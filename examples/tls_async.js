import { Socket } from "k6/x/tcp";

/**
 * TLS connection with async/await example.
 * Shows how to use TLS with async operations for better flow control.
 */

export const options = {
    // Optional: Configure TLS settings
    // insecureSkipTLSVerify: true, // Skip certificate verification (dev/test only)
};

export default async function () {
    const socket = new Socket({
        tags: {
            protocol: "tls-async",
        },
    });

    let fullResponse = "";

    // Set up event handlers
    const dataPromise = new Promise((resolve) => {
        socket.on("data", (data) => {
            const chunk = String.fromCharCode.apply(null, new Uint8Array(data));
            fullResponse += chunk;
            console.log("Received data chunk, length:", data.byteLength);

            // Check if we got the complete response
            if (chunk.includes("</html>")) {
                resolve();
            }
        });
    });

    const closePromise = new Promise((resolve) => {
        socket.on("close", () => {
            console.log("Connection closed");
            resolve();
        });
    });

    socket.on("error", (err) => {
        console.error("Socket error:", err);
    });

    // Connect with TLS
    const host = __ENV.TLS_HOST || "example.com";
    const port = parseInt(__ENV.TLS_PORT || "443");

    await socket.connectAsync({
        port: port,
        host: host,
        tls: true,
    });

    console.log("TLS connection established");

    // Send HTTPS request
    await socket.writeAsync(
        `GET / HTTP/1.1\r\nHost: ${host}\r\nConnection: close\r\n\r\n`
    );

    console.log("Request sent, waiting for response...");

    // Wait for response (with 2 second timeout)
    await Promise.race([
        dataPromise,
        new Promise((resolve) => setTimeout(resolve, 2000))
    ]);

    console.log("Response received, length:", fullResponse.length);
    if (fullResponse.length > 200) {
        console.log("First 200 chars:", fullResponse.substring(0, 200));
    } else {
        console.log("Full response:", fullResponse);
    }

    // Verify it's HTML
    if (fullResponse.includes("HTTP/") && fullResponse.includes("html")) {
        console.log("✓ Successfully received HTTP/1.1 HTML response");
    } else if (fullResponse.length === 0) {
        console.log("✗ No response received (connection may have closed)");
    }

    // Clean up
    socket.destroy();
    await closePromise;
}
