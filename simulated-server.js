const http = require('http');
const server = http.createServer((req, res) => {
  console.log(`[${new Date().toISOString()}] Request: ${req.method} ${req.url}`);
  
  if (req.url === '/') {
    console.log('  -> Handling root route');
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('Welcome to the simulated webserver!');
  } else if (req.url === '/users') {
    console.log('  -> Handling users route');
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ users: ['Alice', 'Bob', 'Charlie'] }));
  } else if (req.url === '/about') {
    console.log('  -> Handling about route');
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('This is a simulated webserver for demonstration.');
  } else if (req.url === '/login') {
    console.log('  -> Handling login route');
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ message: 'Login successful', user: 'example' }));
  } else if (req.url === '/register') {
    console.log('  -> Handling register route');
    res.writeHead(201, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ message: 'Registration complete', userId: Math.floor(Math.random() * 1000) }));
  } else if (req.url === '/profile') {
    console.log('  -> Handling profile route');
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ name: 'John Doe', email: 'john@example.com' }));
  } else if (req.url === '/logout') {
    console.log('  -> Handling logout route');
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('Logged out successfully');
  } else if (req.url === '/dashboard') {
    console.log('  -> Handling dashboard route');
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('Welcome to your dashboard');
  } else {
    console.log('  -> Handling 404 route');
    res.writeHead(404, { 'Content-Type': 'text/plain' });
    res.end('Page not found');
  }
});

server.listen(3001, () => {
  console.log(`[${new Date().toISOString()}] Server started on port 3001`);
  startTrafficSimulation();
});

function startTrafficSimulation() {
  const endpoints = ['/', '/users', '/about', '/login', '/register', '/profile', '/logout', '/dashboard'];
  console.log(`[${new Date().toISOString()}] Starting continuous traffic simulation`);

  setInterval(() => {
    const userId = Math.floor(Math.random() * 10000) + 1; // random user ID 1-10000
    const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
    const options = {
      hostname: 'localhost',
      port: 3001,
      path: endpoint,
      method: 'GET'
    };
    const req = http.request(options, (res) => {
      console.log(`[${new Date().toISOString()}] User ${userId} accessed ${endpoint} - Status: ${res.statusCode}`);
    });
    req.on('error', (e) => {
      console.error(`[${new Date().toISOString()}] Error for User ${userId}: ${e.message}`);
    });
    req.end();
  }, 500 + Math.random() * 2000); // random interval 0.5-2.5s
}
