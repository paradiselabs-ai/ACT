/**
 * PVM embedding sidecar cache tests
 *
 * Exercises the persistence layer of LocalEmbeddingVectorStore without the
 * real transformer model: a subclass fakes embed() deterministically and
 * controls the store's real/hash mode, so the tests cover the sidecar's
 * hit/miss/persist/guard logic in milliseconds.
 */

import { LocalEmbeddingVectorStore } from '../services/LocalEmbeddingVectorStore.js';
import { CoordinationMessage } from '../types/coordination.js';
import fs from 'fs/promises';
import path from 'path';

const TEST_DATA_DIR = './test-data-sidecar';
const SIDECAR_PATH = path.join(TEST_DATA_DIR, 'pvm-vectors.jsonl');

// Deterministic fake embedder: vector derived from text length, no model load.
class FakeEmbedStore extends LocalEmbeddingVectorStore {
  public embedCalls = 0;
  constructor(private fakeMode: 'real' | 'hash' = 'real') {
    super({ sidecarPath: SIDECAR_PATH });
  }
  async embed(text: string): Promise<number[]> {
    this.embedCalls++;
    (this as any).mode = this.fakeMode;
    return [text.length, 1, 0];
  }
}

const mkMessage = (n: number, text = `payload ${n}`): CoordinationMessage => ({
  timestamp: `2026-07-06T01:00:${String(n).padStart(2, '0')}Z`,
  agent: 'dev-1',
  message: text,
  type: 'coordination'
});

describe('PVM sidecar cache', () => {
  beforeEach(async () => {
    await fs.rm(TEST_DATA_DIR, { recursive: true, force: true });
  });

  afterEach(async () => {
    await fs.rm(TEST_DATA_DIR, { recursive: true, force: true });
  });

  it('cold run embeds fresh and persists; warm run serves everything from the sidecar', async () => {
    const cold = new FakeEmbedStore();
    await cold.batchStore([mkMessage(1), mkMessage(2), mkMessage(3)]);
    expect(cold.getSidecarStats()).toEqual({ fromCache: 0, freshEmbeds: 3 });
    expect(cold.embedCalls).toBe(3);

    const lines = (await fs.readFile(SIDECAR_PATH, 'utf-8')).trim().split('\n');
    expect(lines).toHaveLength(3);

    // "Restart": a brand-new store over the same sidecar
    const warm = new FakeEmbedStore();
    await warm.batchStore([mkMessage(1), mkMessage(2), mkMessage(3)]);
    expect(warm.getSidecarStats()).toEqual({ fromCache: 3, freshEmbeds: 0 });
    expect(warm.embedCalls).toBe(0);

    // Vectors are byte-identical to the cold run's
    expect((warm as any).points.map((p: any) => p.vector))
      .toEqual((cold as any).points.map((p: any) => p.vector));
  });

  it('changed text for the same event key is a cache miss (re-embed)', async () => {
    const cold = new FakeEmbedStore();
    await cold.store(mkMessage(1));

    const warm = new FakeEmbedStore();
    await warm.store(mkMessage(1, 'entirely different text'));
    expect(warm.getSidecarStats()).toEqual({ fromCache: 0, freshEmbeds: 1 });
    expect(warm.embedCalls).toBe(1);
  });

  it('hash-mode embeddings are never persisted', async () => {
    const hashStore = new FakeEmbedStore('hash');
    await hashStore.batchStore([mkMessage(1), mkMessage(2)]);
    expect(hashStore.getSidecarStats().freshEmbeds).toBe(2);
    await expect(fs.access(SIDECAR_PATH)).rejects.toThrow(); // no file written

    // After the model "heals", a real-mode store re-embeds those events
    const healed = new FakeEmbedStore('real');
    await healed.store(mkMessage(1));
    expect(healed.getSidecarStats()).toEqual({ fromCache: 0, freshEmbeds: 1 });
    expect((await fs.readFile(SIDECAR_PATH, 'utf-8')).trim().split('\n')).toHaveLength(1);
  });

  it('tolerates a truncated final sidecar line', async () => {
    const cold = new FakeEmbedStore();
    await cold.store(mkMessage(1));
    await fs.appendFile(SIDECAR_PATH, '{"key":"2026-07-06T01:00:02Z_dev', 'utf-8');

    const warm = new FakeEmbedStore();
    await warm.store(mkMessage(1));
    expect(warm.getSidecarStats()).toEqual({ fromCache: 1, freshEmbeds: 0 });
  });

  it('explicitly provided embeddings bypass the sidecar entirely', async () => {
    const store = new FakeEmbedStore();
    await store.store(mkMessage(1), [9, 9, 9]);
    expect(store.getSidecarStats()).toEqual({ fromCache: 0, freshEmbeds: 0 });
    await expect(fs.access(SIDECAR_PATH)).rejects.toThrow();
  });
});
