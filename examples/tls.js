import { Socket } from "k6/x/tcp";

/**
 * TLS/SSL connection example.
 * Demonstrates secure TCP connections with TLS encryption.
 * 
 * TLS configuration is handled by k6's standard TLS settings.
 * See: https://grafana.com/docs/k6/latest/using-k6/protocols/ssl-tls/
 */

export const options = {
    // Optional: Skip TLS verification (not recommended for production)
    // insecureSkipTLSVerify: true,
};

export default function () {
    const socket = new Socket({
        tags: {
            protocol: "tls",
        },
    });

    socket.on("connect", () => {
        console.log("Secure TLS connection established");
        socket.write("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n");
    });

    socket.on("data", (data) => {
        const response = String.fromCharCode.apply(null, new Uint8Array(data));
        console.log("Received encrypted response:", response.substring(0, 200));
        socket.destroy();
    });

    socket.on("close", () => {
        console.log("Secure connection closed");
    });

    socket.on("error", (err) => {
        console.error("Socket error:", err);
    });

    // Connect with TLS enabled
    const host = __ENV.TLS_HOST || "example.com";
    const port = parseInt(__ENV.TLS_PORT || "443");

    socket.connect({
        port: parseInt(port),
        host: host,
        tls: true, // Enable TLS encryption
    });
}
