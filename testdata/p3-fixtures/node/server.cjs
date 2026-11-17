'use strict';

const http = require('node:http');

function createServer({ release = 'fixture-release', ready = true } = {}) {
  return http.createServer((req, res) => {
    if (req.method !== 'GET') {
      res.writeHead(405, { Allow: 'GET', 'Content-Type': 'text/plain' });
      return res.end('method not allowed');
    }
    if (req.url === '/healthz') {
      res.writeHead(ready ? 200 : 503, { 'Content-Type': 'application/json' });
      return res.end(JSON.stringify({ ready }));
    }
    if (req.url === '/') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      return res.end(JSON.stringify({ release }));
    }
    res.writeHead(404, { 'Content-Type': 'text/plain' });
    res.end('not found');
  });
}

module.exports = { createServer };
