import { Socket } from "k6/x/tcp";

/**
 * Socket options example demonstrating tags for metrics.
 * Tags are useful for organizing and filtering metrics in k6 output.
 */
export default function () {
    // Create socket with tags for metrics
    const socket = new Socket({
        tags: {
            endpoint: "api-server",
            environment: "production",
            protocol: "custom"
        }
    });

    socket.on("connect", () => {
        console.log("Connected with tagged metrics");

        // Write with additional tags specific to this operation
        socket.write("GET /status", {
            tags: {
                operation: "health-check",
                method: "GET"
            }
        });
    });

    socket.on("data", (data) => {
        const response = String.fromCharCode.apply(null, new Uint8Array(data));
        console.log("Status response:", response);
        socket.destroy();
    });

    socket.on("close", () => {
        console.log("Connection closed");
    });

    socket.on("error", (err) => {
        console.error("Socket error:", err);
    });

    // Connect with connection-specific tags
    const host = __ENV.TCP_ECHO_HOST || "localhost";
    const port = __ENV.TCP_ECHO_PORT || "8080";

    socket.connect({
        port: port,
        host: host,
        tags: {
            connection_type: "direct"
        }
    });
}
