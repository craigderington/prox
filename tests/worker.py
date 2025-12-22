#!/usr/bin/env python3
"""
tui_worker.py — Single-file "binary-style" TUI worker spawner + redraw torture test.
- Spawns N simulated worker processes (multiprocessing)
- Parent draws a top/htop-like TUI using curses
- Includes Unicode glyphs, sparklines, progress bars, and a journalctl-ish log pane
- "Worst case" mode redraws EVERYTHING every tick

Controls:
  q  quit
  +/- increase/decrease spawn count (live)
  w  toggle worst-case redraw (more brutal)
  l  toggle log spam
  h  toggle help footer
"""

from __future__ import annotations

import curses
import math
import os
import random
import signal
import sys
import time
from collections import deque
from dataclasses import dataclass
from multiprocessing import Event, Process, Queue, Value
from multiprocessing.sharedctypes import Synchronized
from typing import Deque, List, Tuple

SPARK_CHARS = "▁▂▃▄▅▆▇█"
BAR_CHARS = ("░", "█")
GLYPHS = ["⚙", "⏱", "⛓", "🧠", "📦", "🔧", "🛰", "🧪", "🧯", "🧲", "🗄", "🧵"]
STATE_GLYPHS = {
    "RUN": "▶",
    "SLEEP": "⏾",
    "IO": "⇄",
    "ZOMB": "✖",
    "WAIT": "…",
}

def clamp(x: float, lo: float, hi: float) -> float:
    return lo if x < lo else hi if x > hi else x

def sparkline(values: List[float], width: int) -> str:
    if width <= 0:
        return ""
    if not values:
        return " " * width
    vals = values[-width:]
    lo, hi = min(vals), max(vals)
    if hi - lo < 1e-9:
        return SPARK_CHARS[0] * len(vals) + " " * (width - len(vals))
    out = []
    for v in vals:
        t = (v - lo) / (hi - lo)
        idx = int(clamp(t, 0.0, 0.999999) * len(SPARK_CHARS))
        out.append(SPARK_CHARS[idx])
    return "".join(out).ljust(width)

def bar(pct: float, width: int) -> str:
    if width <= 0:
        return ""
    pct = clamp(pct, 0.0, 1.0)
    filled = int(pct * width)
    return (BAR_CHARS[1] * filled + BAR_CHARS[0] * (width - filled))[:width]

def fmt_bytes(n: float) -> str:
    units = ["B", "K", "M", "G", "T"]
    u = 0
    while n >= 1024 and u < len(units) - 1:
        n /= 1024.0
        u += 1
    if u == 0:
        return f"{int(n)}{units[u]}"
    return f"{n:0.1f}{units[u]}"

def now_ts() -> str:
    return time.strftime("%H:%M:%S")

@dataclass
class ProcStat:
    pid: int
    name: str
    state: str
    cpu: float
    mem: float
    io: float
    thr: int
    prog: float
    hist_cpu: Deque[float]
    hist_io: Deque[float]
    glyph: str

def worker_main(idx: int, alive: Synchronized, q: Queue, stop: Event) -> None:
    """
    Simulated process.
    Pushes periodic updates to the parent via Queue.
    Does small CPU bursts + random sleep to resemble activity.
    """
    random.seed(os.getpid() ^ int(time.time()))
    name = f"simproc-{idx:03d}"
    glyph = random.choice(GLYPHS)
    state = "RUN"
    prog = random.random()
    mem_base = random.uniform(10.0, 350.0)  # MB-ish
    io_base = random.uniform(0.0, 8.0)

    # Help parent track count reliably even if processes crash.
    with alive.get_lock():
        alive.value += 1

    t0 = time.time()
    try:
        while not stop.is_set():
            t = time.time() - t0

            # State machine-ish
            r = random.random()
            if r < 0.70:
                state = "RUN"
            elif r < 0.85:
                state = "SLEEP"
            elif r < 0.97:
                state = "IO"
            else:
                state = "WAIT"

            # Simulated metrics
            cpu = clamp(50.0 * (0.6 + 0.4 * math.sin(t * random.uniform(0.8, 2.0) + idx)), 0.0, 99.9)
            mem = clamp(mem_base + 60.0 * math.sin(t * 0.6 + idx / 7.0) + random.uniform(-8, 8), 1.0, 1024.0)
            io = clamp(io_base + 4.0 * abs(math.sin(t * 1.3 + idx)) + random.uniform(0, 2), 0.0, 50.0)
            thr = int(clamp(1 + 12 * abs(math.sin(t * 0.9 + idx / 3.0)) + random.randint(0, 3), 1, 64))
            prog = (prog + random.uniform(0.002, 0.02)) % 1.0

            # Cheap CPU "work" burst to feel real without melting the machine
            if state == "RUN":
                acc = 0.0
                for _ in range(random.randint(600, 2200)):
                    acc += math.sqrt(random.random())
                _ = acc  # keep optimizer honest

            q.put((os.getpid(), name, state, cpu, mem, io, thr, prog, glyph, time.time()))
            # Timing: mix of small sleeps like sched jitter
            if state == "SLEEP":
                time.sleep(random.uniform(0.10, 0.45))
            elif state == "IO":
                time.sleep(random.uniform(0.03, 0.18))
            else:
                time.sleep(random.uniform(0.02, 0.12))
    finally:
        with alive.get_lock():
            alive.value -= 1

def safe_addstr(win, y: int, x: int, s: str, attr: int = 0) -> None:
    try:
        h, w = win.getmaxyx()
        if y < 0 or y >= h:
            return
        if x < 0:
            s = s[-x:]
            x = 0
        if x >= w:
            return
        win.addnstr(y, x, s, w - x, attr)
    except curses.error:
        pass

def init_colors() -> None:
    if not curses.has_colors():
        return
    curses.start_color()
    curses.use_default_colors()
    # A handful of pairs, we'll rotate to stress color rendering
    pairs = [
        (curses.COLOR_CYAN, -1),
        (curses.COLOR_GREEN, -1),
        (curses.COLOR_YELLOW, -1),
        (curses.COLOR_MAGENTA, -1),
        (curses.COLOR_RED, -1),
        (curses.COLOR_WHITE, -1),
        (curses.COLOR_BLUE, -1),
    ]
    for i, (fg, bg) in enumerate(pairs, start=1):
        curses.init_pair(i, fg, bg)

def color(i: int) -> int:
    if not curses.has_colors():
        return 0
    i = 1 + (i % 7)
    return curses.color_pair(i)

def draw_box(win, y0: int, x0: int, y1: int, x1: int, attr: int = 0) -> None:
    # minimal box using unicode
    tl, tr, bl, br = "┌", "┐", "└", "┘"
    hz, vt = "─", "│"
    safe_addstr(win, y0, x0, tl + hz * max(0, x1 - x0 - 1) + tr, attr)
    for y in range(y0 + 1, y1):
        safe_addstr(win, y, x0, vt, attr)
        safe_addstr(win, y, x1, vt, attr)
    safe_addstr(win, y1, x0, bl + hz * max(0, x1 - x0 - 1) + br, attr)

def main(stdscr) -> int:
    curses.curs_set(0)
    stdscr.nodelay(True)
    stdscr.timeout(0)
    init_colors()

    q: Queue = Queue(maxsize=5000)
    stop = Event()
    alive = Value("i", 0)

    # Runtime toggles
    worst_case = True
    log_spam = True
    show_help = True

    # Stats storage
    procs: dict[int, ProcStat] = {}
    logs: Deque[str] = deque(maxlen=5000)
    # Global history
    hist_total_cpu: Deque[float] = deque(maxlen=200)
    hist_total_io: Deque[float] = deque(maxlen=200)

    def spawn(n: int) -> List[Process]:
        kids: List[Process] = []
        base = len(procs)
        for i in range(n):
            p = Process(target=worker_main, args=(base + i, alive, q, stop), daemon=True)
            p.start()
            kids.append(p)
        return kids

    kids: List[Process] = []
    target_n = int(os.environ.get("N", "24"))
    target_n = max(1, min(target_n, 400))
    kids += spawn(target_n)

    last_draw = 0.0
    last_log = 0.0
    fps = float(os.environ.get("FPS", "20"))
    fps = clamp(fps, 5.0, 60.0)
    dt = 1.0 / fps

    # Make Ctrl-C clean
    def handle_sig(sig, frame):
        stop.set()
    signal.signal(signal.SIGINT, handle_sig)
    signal.signal(signal.SIGTERM, handle_sig)

    while not stop.is_set():
        # Drain queue (lots)
        drained = 0
        while True:
            try:
                (pid, name, state, cpu, mem, io, thr, prog, glyph, ts) = q.get_nowait()
            except Exception:
                break
            drained += 1
            ps = procs.get(pid)
            if ps is None:
                ps = ProcStat(
                    pid=pid,
                    name=name,
                    state=state,
                    cpu=cpu,
                    mem=mem,
                    io=io,
                    thr=thr,
                    prog=prog,
                    hist_cpu=deque(maxlen=120),
                    hist_io=deque(maxlen=120),
                    glyph=glyph,
                )
                procs[pid] = ps
                logs.append(f"{now_ts()} {pid} {name}: started {glyph}")
            ps.state = state
            ps.cpu = cpu
            ps.mem = mem
            ps.io = io
            ps.thr = thr
            ps.prog = prog
            ps.glyph = glyph
            ps.hist_cpu.append(cpu)
            ps.hist_io.append(io)

            # journalctl-like log spam
            if log_spam and (time.time() - last_log) > 0.03 and random.random() < 0.55:
                lvl = random.choice(["INFO", "WARN", "DEBUG", "NOTICE", "ERR"])
                msg = random.choice([
                    "tick", "alloc", "gc", "io", "net", "sched", "lock", "wake", "sleep", "burst", "flush", "rotate"
                ])
                logs.append(f"{now_ts()} {lvl} {name}[{pid}]: {msg} cpu={cpu:0.1f}% mem={mem:0.1f} io={io:0.1f}")
                last_log = time.time()

        # Maintain target spawn count (simple)
        if alive.value < target_n:
            kids += spawn(target_n - alive.value)
        elif alive.value > target_n:
            # Ask a few to stop by setting global stop? (too aggressive)
            # Instead: temporarily lower load by turning off log spam and letting extra die naturally is not right.
            # We'll "soft" reduce by sending SIGTERM to some children.
            extra = alive.value - target_n
            killed = 0
            for ps in list(procs.values()):
                if killed >= extra:
                    break
                try:
                    os.kill(ps.pid, signal.SIGTERM)
                    logs.append(f"{now_ts()} NOTICE manager: SIGTERM pid={ps.pid}")
                    killed += 1
                except Exception:
                    pass

        # Input
        try:
            ch = stdscr.getch()
        except Exception:
            ch = -1
        if ch != -1:
            if ch in (ord("q"), ord("Q")):
                stop.set()
                break
            elif ch == ord("+"):
                target_n = min(400, target_n + 5)
                logs.append(f"{now_ts()} NOTICE manager: target_n={target_n}")
            elif ch == ord("-"):
                target_n = max(1, target_n - 5)
                logs.append(f"{now_ts()} NOTICE manager: target_n={target_n}")
            elif ch in (ord("w"), ord("W")):
                worst_case = not worst_case
                logs.append(f"{now_ts()} NOTICE manager: worst_case={'on' if worst_case else 'off'}")
            elif ch in (ord("l"), ord("L")):
                log_spam = not log_spam
                logs.append(f"{now_ts()} NOTICE manager: log_spam={'on' if log_spam else 'off'}")
            elif ch in (ord("h"), ord("H")):
                show_help = not show_help

        # Draw at fps
        now = time.time()
        if now - last_draw < dt:
            time.sleep(0.001)
            continue
        last_draw = now

        h, w = stdscr.getmaxyx()
        if h < 12 or w < 60:
            stdscr.erase()
            safe_addstr(stdscr, 0, 0, "Resize terminal (need ~60x12). q=quit")
            stdscr.refresh()
            continue

        # Compute aggregates
        plist = sorted(procs.values(), key=lambda p: (p.cpu, p.io, p.mem), reverse=True)
        topn = plist[: max(1, h - 10)]
        total_cpu = sum(p.cpu for p in plist[: min(len(plist), 80)]) / max(1, min(len(plist), 80))
        total_io = sum(p.io for p in plist[: min(len(plist), 80)]) / max(1, min(len(plist), 80))
        total_mem = sum(p.mem for p in plist[: min(len(plist), 200)])
        hist_total_cpu.append(total_cpu)
        hist_total_io.append(total_io)

        # worst-case: full erase, redraw everything
        if worst_case:
            stdscr.erase()

        # Header (top-like)
        title = f"tui_worker  pid={os.getpid()}  procs={alive.value}/{target_n}  drained={drained}  fps={fps:0.0f}  {time.strftime('%Y-%m-%d %H:%M:%S')}"
        safe_addstr(stdscr, 0, 0, title.ljust(w), curses.A_BOLD | color(int(now * 10)))

        # Spark + bars line
        line1 = f"CPU {bar(total_cpu/100.0, 20)} {total_cpu:5.1f}%   IO {bar(total_io/50.0, 20)} {total_io:5.1f}   MEM {fmt_bytes(total_mem*1024*1024):>7}"
        safe_addstr(stdscr, 1, 0, line1.ljust(w), color(int(now * 7)))

        spw = min(40, max(10, w - 2))
        safe_addstr(stdscr, 2, 0, f"cpu {sparkline(list(hist_total_cpu), spw)}  io {sparkline(list(hist_total_io), spw)}".ljust(w), color(int(now * 3)))

        # Boxes like panels
        mid = max(40, w // 2)
        proc_box_top = 3
        proc_box_bot = h - 5
        log_box_top = 3
        log_box_bot = h - 2

        draw_box(stdscr, proc_box_top, 0, proc_box_bot, mid - 1, color(1))
        draw_box(stdscr, log_box_top, mid, log_box_bot, w - 1, color(2))

        safe_addstr(stdscr, proc_box_top, 2, " PROCESS LIST ", curses.A_BOLD | color(3))
        safe_addstr(stdscr, log_box_top, mid + 2, " JOURNAL ", curses.A_BOLD | color(4))

        # Process table header (htop-ish columns)
        cols = " PID  S  CPU%   MEM(MB)   IO   THR  PROG  NAME               CPU spark"
        safe_addstr(stdscr, proc_box_top + 1, 1, cols[: mid - 3].ljust(mid - 3), curses.A_BOLD | color(5))

        # Rows
        max_rows = (proc_box_bot - (proc_box_top + 2))
        for i, p in enumerate(topn[:max_rows]):
            y = proc_box_top + 2 + i
            state = STATE_GLYPHS.get(p.state, "?")
            progbar = bar(p.prog, 8)
            sp = sparkline(list(p.hist_cpu), 10)
            row = f"{p.pid:5d} {state} {p.cpu:5.1f}  {p.mem:8.1f}  {p.io:5.1f} {p.thr:4d}  {progbar}  {p.glyph} {p.name:<18} {sp}"
            attr = color(p.pid) | (curses.A_BOLD if p.cpu > 70 else 0)
            safe_addstr(stdscr, y, 1, row[: mid - 3].ljust(mid - 3), attr)

        # Log pane (journalctl-ish: newest at bottom)
        log_h = (log_box_bot - (log_box_top + 1)) - 1
        log_w = (w - 1) - (mid + 1)
        recent = list(logs)[-log_h:]
        start_y = log_box_top + 1
        for i in range(log_h):
            y = start_y + i
            msg = recent[i] if i < len(recent) else ""
            # subtle "levels" highlight
            attr = color(i + int(now * 2))
            if " ERR " in msg or "ERR" in msg:
                attr |= curses.A_BOLD
            safe_addstr(stdscr, y, mid + 1, msg[: log_w].ljust(log_w), attr)

        # Footer
        if show_help:
            footer = "q quit | +/- procs | w worst-case | l log spam | h help"
        else:
            footer = f"mode={'WORST' if worst_case else 'normal'} logs={'on' if log_spam else 'off'}  (+/- to change procs)"
        safe_addstr(stdscr, h - 1, 0, footer.ljust(w), curses.A_REVERSE | color(int(now)))

        stdscr.refresh()

    # Cleanup
    stop.set()
    for p in kids:
        try:
            p.join(timeout=0.15)
        except Exception:
            pass
    return 0

if __name__ == "__main__":
    # Provide stable behavior in weird terminals
    os.environ.setdefault("ESCDELAY", "25")
    try:
        raise SystemExit(curses.wrapper(main))
    except KeyboardInterrupt:
        raise SystemExit(0)

