import assert from "node:assert";
import test from "node:test";
import { QueueController } from '../spotify/queueController';

test('Queue controller test 1', (t) => {
  // This test passes because it does not throw an exception.
  const qc = new QueueController(99999999);
  qc.add('Дора - Дорадура', 1288423, '11111');
  qc.add('Дора - Осень пьяная', 1521612, '22222');
  qc.add('Мейби Бейби - Охегао', 5412325, '33333');

  const actualQueue = qc.getFrom('22222');

  assert.strictEqual(actualQueue.length, 1);
  assert.strictEqual(actualQueue[0].uri, '33333');
});

test('Queue controller test queue end', (t) => {
  // This test passes because it does not throw an exception.
  const qc = new QueueController(99999999);
  qc.add('Дора - Дорадура', 1288423, '11111');
  qc.add('Дора - Осень пьяная', 1521612, '22222');
  qc.add('Мейби Бейби - Охегао', 5412325, '33333');

  const actualQueue = qc.getFrom('4444');

  assert.strictEqual(actualQueue.length, 0);
});

test('Queue controller test 2', (t) => {
  // This test passes because it does not throw an exception.
  const qc = new QueueController(99999999);
  qc.add('Дора - Дорадура', 1288423, '11111');
  qc.add('Дора - Осень пьяная', 1521612, '22222');
  qc.add('Мейби Бейби - Охегао', 5412325, '33333');

  const actualQueue = qc.getFrom('11111');

  assert.strictEqual(actualQueue.length, 2);
  assert.strictEqual(actualQueue[0].uri, '22222');
  assert.strictEqual(actualQueue[1].uri, '33333');
});