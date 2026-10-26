const fs = require('fs');
const src = fs.readFileSync(__dirname + '/app.js', 'utf8');
const start = src.indexOf('function sshPortValue');
const end = src.indexOf('const ND_TOKEN');
if (start < 0 || end < 0) { console.error('helpers missing'); process.exit(1); }
eval(src.slice(start, end));

function assert(cond, msg) { if (!cond) { console.error(msg); process.exit(1); } }

let body = {};
assert(applySshPort(body, '') === true, 'empty should allow submit');
assert(!('primary_ssh_port' in body), 'empty must omit key');

body = {};
assert(applySshPort(body, '52222') === true, 'valid should allow submit');
assert(body.primary_ssh_port === 52222, 'valid must set number');

body = {};
assert(applySshPort(body, ' 2200 ') === true, 'trimmed valid');
assert(body.primary_ssh_port === 2200, 'trimmed value');

for (const bad of ['0', '-1', 'abc', '22abc', '65536']) {
  body = {};
  assert(applySshPort(body, bad) === false, bad + ' must block submit');
  assert(!('primary_ssh_port' in body), bad + ' must not set key');
}
console.log('web ssh port payload ok');
