// Pure event bookkeeping for a live session: the send outbox, replay dedupe
// and the per-session sequence tracker. Nothing here touches a socket, the
// network or the DOM, so it can be reasoned about and tested on its own
// (#296).

import type { RiffpadEvent } from "./types";

// Outbox keeps outgoing events that could not be written while the socket was
// down. Items are flushed in order after the next successful hello, so nothing
// the user typed is silently lost. Events that were already written to the
// socket are never queued, which avoids duplicate execution on replay.
export class Outbox<T> {
  private items: T[] = [];

  push(item: T) {
    this.items.push(item);
  }

  drain(): T[] {
    const out = this.items;
    this.items = [];
    return out;
  }

  clear() {
    this.items = [];
  }

  get size() {
    return this.items.length;
  }
}

// dedupeEvent returns true when ev has already been seen (replay after a
// reconnect). Exported for tests.
export function dedupeEvent(seen: Set<string>, ev: RiffpadEvent): boolean {
  if (seen.has(ev.id)) return true;
  seen.add(ev.id);
  return false;
}

// MAX_SEEN_IDS bounds the dedupe set: a session open for days would otherwise
// grow it without limit (#174). JS Sets iterate in insertion order, so the
// oldest ids are evicted first; reconnect replays only cover recent history,
// far below this bound, so dedupe accuracy is unaffected in practice.
export const MAX_SEEN_IDS = 5000;

// dedupeBounded is dedupeEvent plus the size bound. Exported for tests.
export function dedupeBounded(seen: Set<string>, ev: RiffpadEvent): boolean {
  if (dedupeEvent(seen, ev)) return true;
  if (seen.size > MAX_SEEN_IDS) {
    const it = seen.values();
    for (let i = seen.size - MAX_SEEN_IDS; i > 0; i--) {
      seen.delete(it.next().value as string);
    }
  }
  return false;
}

// SeqTracker detects holes in the per-session event sequence (#173). Send
// buffers that overflow drop messages, so a client that only counts what
// arrives can never notice a missing approval card. Events carry an
// increasing per-session seq; a jump means something was lost in between.
//
// Replay and live events can interleave on one connection (the daemon queues
// the replay burst while the pump keeps broadcasting), so a hole may be
// filled by messages that simply arrive later in the same stream: note()
// only records the gap, and pendingGap() reports it once the drain finished
// with the hole still open. Exported for tests.
export class SeqTracker {
  private lastSeq = 0;
  private gapFloor = 0;
  private gapCeil = 0;
  private gapMissing = 0;

  note(seq: number | undefined): void {
    if (!seq || seq <= 0) return; // no seq (old daemon): skip detection
    if (seq > this.lastSeq) {
      if (this.lastSeq > 0 && seq > this.lastSeq + 1) {
        if (this.gapMissing === 0) this.gapFloor = this.lastSeq + 1;
        this.gapCeil = seq - 1;
        this.gapMissing += seq - this.lastSeq - 1;
      }
      this.lastSeq = seq;
    } else if (this.gapMissing > 0 && seq >= this.gapFloor && seq <= this.gapCeil) {
      this.gapMissing--;
    }
  }

  // pendingGap returns the still-unfilled hole, if any.
  pendingGap(): { floor: number; ceil: number; missing: number } | null {
    if (this.gapMissing <= 0) return null;
    return { floor: this.gapFloor, ceil: this.gapCeil, missing: this.gapMissing };
  }

  clearGap(): void {
    this.gapMissing = 0;
    this.gapFloor = 0;
    this.gapCeil = 0;
  }
}
