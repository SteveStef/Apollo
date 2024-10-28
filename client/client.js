const net = require('net');
const { Buffer } = require('buffer');

const port = 4000;
const host = '127.0.0.1';
const api_key = "penguins";

const client = net.createConnection({ host: host, port: port }, () => {
  console.log(`Connected to server on ${host}:${port}`);
  client.write(Buffer.from(api_key)); // Authenticate once per session
});

client.on('data', (data) => {
  console.log('Response from server:', data.toString());
});

client.on('end', () => {
  console.log('Disconnected from server');
});

client.on('error', (err) => {
  console.error(`Connection error: ${err.message}`);
});

export function set(key, val, ttl) {
  client.write(Buffer.from(api_key));
  client.write(Buffer.from("SET"));

  let lengthBuffer = Buffer.alloc(4);
  lengthBuffer.writeUInt32BE(key.length);
  client.write(lengthBuffer);

  client.write(Buffer.from(key));

  lengthBuffer = Buffer.alloc(4);
  lengthBuffer.writeUInt32BE(val.length);
  client.write(lengthBuffer);

  client.write(Buffer.from(val));

  const ttlBuffer = Buffer.alloc(4);
  ttlBuffer.writeUInt32BE(parseInt(ttl, 10));
  client.write(ttlBuffer);
}

export function get(key) {
  client.write(Buffer.from(api_key));
  client.write(Buffer.from("GET"));

  let lengthBuffer = Buffer.alloc(4);
  lengthBuffer.writeUInt32BE(key.length);
  client.write(lengthBuffer);

  client.write(Buffer.from(key));
}

export function del(key) {
  client.write(Buffer.from(api_key));
  client.write(Buffer.from("DEL"));

  let lengthBuffer = Buffer.alloc(4);
  lengthBuffer.writeUInt32BE(key.length);
  client.write(lengthBuffer);

  client.write(Buffer.from(key));
}

export function ral() {
  client.write(Buffer.from(api_key));
  client.write(Buffer.from("RAL"));
}

set("foo", "bar", "10");
get("foo");
del("foo");
ral();

setTimeout(() => {
  client.end();
}, 5000);

