const http = require('http');

const server = http.createServer((req, res) => {
  console.log(`[${new Date().toISOString()}] ${req.method} ${req.url}`);
  res.writeHead(200, { 'Content-Type': 'text/plain' });
  res.end('Hello from Node.js app!\n');
});

const port = process.env.PORT || 3000;

server.listen(port, () => {
  console.log(`Node.js server running on port ${port}`);
  console.log(`PID: ${process.pid}`);
}).on('error', (err) => {
  if (err.code === 'EADDRINUSE') {
    console.error(`Port ${port} is already in use. Try setting PORT environment variable to a different port.`);
    console.error(`Example: PORT=8081 prox start server.js --name api-server`);
    process.exit(1);
  } else {
    console.error('Server error:', err);
    process.exit(1);
  }
});