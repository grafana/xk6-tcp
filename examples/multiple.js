import { Socket } from "k6/x/tcp";

/**
 * Multiple connections example.
 * Demonstrates managing multiple concurrent TCP connections.
 */
export default async function () {
    const host = __ENV.TCP_ECHO_HOST || "localhost";
    const port = __ENV.TCP_ECHO_PORT || "8080";
    const connectionCount = 3;

    const connections = [];

    // Create multiple connections
    for (let i = 0; i < connectionCount; i++) {
        const socket = new Socket({
            tags: {
                connection_id: `conn-${i}`
            }
        });

        const connectionPromise = new Promise((resolve, reject) => {
            socket.on("connect", () => {
                console.log(`Connection ${i} established`);
                socket.write(`Hello from connection ${i}`);
            });

            socket.on("data", (data) => {
                const response = String.fromCharCode.apply(null, new Uint8Array(data));
                console.log(`Connection ${i} received:`, response);
                socket.destroy();
            });

            socket.on("close", () => {
                console.log(`Connection ${i} closed`);
                resolve();
            });

            socket.on("error", (err) => {
                console.error(`Connection ${i} error:`, err);
                reject(err);
            });

            socket.connect(port, host);
        });

        connections.push(connectionPromise);
    }

    // Wait for all connections to complete
    await Promise.all(connections);
    console.log(`All ${connectionCount} connections completed`);
}
