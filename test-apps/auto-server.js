const http = require('http');

const server = http.createServer((req, res) => {
  const startTime = Date.now();
  const timestamp = new Date().toISOString();

  const responseBody = 'Hello from Auto-Port Node.js app!\n';
  const contentLength = Buffer.byteLength(responseBody);

  res.on('finish', () => {
    const responseTime = Date.now() - startTime;
    const statusCode = res.statusCode;

    // Apache-style combined log format (simplified)
    console.log(`${new Date().toISOString()} ${req.socket.remoteAddress || '127.0.0.1'} "${req.method} ${req.url} HTTP/${req.httpVersion}" ${statusCode} ${contentLength} ${responseTime}ms "${req.headers['user-agent'] || ''}"`);
  });

  res.writeHead(200, {
    'Content-Type': 'text/plain',
    'Content-Length': contentLength
  });
  res.end('Hello from Auto-Port Node.js app!\n');
});

// Try to find an available port starting from the preferred port
function findAvailablePort(startPort, callback) {
  const server = http.createServer();

  server.listen(startPort, () => {
    const port = server.address().port;
    server.close(() => callback(null, port));
  });

  server.on('error', (err) => {
    if (err.code === 'EADDRINUSE') {
      // Port is in use, try the next one
      findAvailablePort(startPort + 1, callback);
    } else {
      callback(err);
    }
  });
}

const preferredPort = parseInt(process.env.PORT) || 3000;

findAvailablePort(preferredPort, (err, port) => {
  if (err) {
    console.error('Failed to find available port:', err);
    process.exit(1);
  }

  server.listen(port, () => {
    console.log(`Node.js server running on port ${port} (auto-selected)`);
    console.log(`PID: ${process.pid}`);
    if (port !== preferredPort) {
      console.log(`Note: Preferred port ${preferredPort} was in use, using ${port} instead`);
    }
  }).on('error', (err) => {
    console.error('Server error:', err);
    process.exit(1);
  });
});