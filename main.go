// snake is a snake game disguised as a Spring Boot log tail.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

const (
	boardW, boardH = 16, 10
	defaultTick    = 150 * time.Millisecond
	bootTick       = 100 * time.Millisecond // boot log pace, independent of game speed
	trafficEvery   = 3                      // 1-in-3 ticks push a traffic line
	initialLen     = 3
	minCols        = 119 // one full board line
	minRows        = boardH + 6

	causeWall = "NullPointerException"
	causeSelf = "StackOverflowError"
	causeFull = "OutOfMemoryError"
)

type point struct{ x, y int }

var (
	up    = point{0, -1}
	down  = point{0, 1}
	left  = point{-1, 0}
	right = point{1, 0}
)

type game struct {
	snake []point // tail first; head = snake[len-1]
	dir   point   // direction last moved
	next  point   // direction requested by input, applied on next step
	food  point
	score int
	dead  bool
	cause string // exception class name for the stack trace once dead
}

// newGame leaves next unset so the snake waits for the first key instead of running into
// the wall while the player is still reading the boot log.
func newGame() *game {
	g := &game{dir: right}
	for i := initialLen - 1; i >= 0; i-- {
		g.snake = append(g.snake, point{boardW/2 - i, boardH / 2})
	}
	g.placeFood()
	return g
}

func (g *game) head() point { return g.snake[len(g.snake)-1] }

func (g *game) occupied(p point) bool {
	for _, s := range g.snake {
		if s == p {
			return true
		}
	}
	return false
}

// turn checks against dir, not next, so two keys in one tick cannot reverse.
func (g *game) turn(d point) {
	if d.x == -g.dir.x && d.y == -g.dir.y {
		return
	}
	g.next = d
}

func (g *game) step() {
	if g.next == (point{}) {
		return
	}
	g.dir = g.next
	h := point{g.head().x + g.dir.x, g.head().y + g.dir.y}
	if h.x < 0 || h.x >= boardW || h.y < 0 || h.y >= boardH {
		g.dead, g.cause = true, causeWall
		return
	}
	for _, s := range g.snake[1:] { // tail cell vacates this step, so it is legal
		if s == h {
			g.dead, g.cause = true, causeSelf
			return
		}
	}
	g.snake = append(g.snake, h)
	if h == g.food {
		g.score++
		g.placeFood()
	} else {
		g.snake = g.snake[1:]
	}
}

func (g *game) placeFood() {
	var free []point
	for y := 0; y < boardH; y++ {
		for x := 0; x < boardW; x++ {
			if p := (point{x, y}); !g.occupied(p) {
				free = append(free, p)
			}
		}
	}
	if len(free) == 0 {
		g.dead, g.cause = true, causeFull
		return
	}
	g.food = free[rand.IntN(len(free))]
}

func (g *game) row(y int) string {
	cells := make([]string, boardW)
	for x := range cells {
		switch p := (point{x, y}); {
		case g.occupied(p):
			cells[x] = "#"
		case p == g.food:
			cells[x] = "@"
		default:
			cells[x] = "."
		}
	}
	return strings.Join(cells, " ")
}

type key struct {
	dir   point
	quit  bool
	pause bool
}

func readKeys(keys chan<- key) {
	var buf [8]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if err != nil {
			keys <- key{quit: true}
			return
		}
		// ponytail: no Esc-timeout disambiguation; arrows arrive as one 3-byte read.
		if n >= 3 && buf[0] == 0x1b && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				keys <- key{dir: up}
			case 'B':
				keys <- key{dir: down}
			case 'C':
				keys <- key{dir: right}
			case 'D':
				keys <- key{dir: left}
			}
			continue
		}
		switch buf[0] {
		case 'q', 'Q', 0x1b, 0x03: // q, Esc, Ctrl-C (raw mode delivers it as a byte)
			keys <- key{quit: true}
		case ' ':
			keys <- key{pause: true}
		case 'w', 'k':
			keys <- key{dir: up}
		case 's', 'j':
			keys <- key{dir: down}
		case 'a', 'h':
			keys <- key{dir: left}
		case 'd', 'l':
			keys <- key{dir: right}
		}
	}
}

func push(ring []string, capacity int, lines ...string) []string {
	ring = append(ring, lines...)
	if len(ring) > capacity {
		ring = ring[len(ring)-capacity:]
	}
	return ring
}

// render writes one frame: cursor home, every line clipped to cols (a wrapped line would
// push the frame past the screen and scroll it), no newline after the last line for the
// same reason, then clear what is left of the screen.
func render(w io.Writer, ring []string, g *game, showBoard bool, cols int) {
	lines := append([]string(nil), ring...)
	if showBoard {
		now := time.Now()
		for y := 0; y < boardH; y++ {
			lines = append(lines, boardLine(now, y, g.row(y)))
		}
	}
	var b strings.Builder
	b.WriteString("\x1b[H")
	for i, l := range lines {
		if len(l) > cols {
			l = l[:cols]
		}
		if i > 0 {
			b.WriteString("\r\n")
		}
		b.WriteString(l + "\x1b[K")
	}
	b.WriteString("\x1b[J")
	io.WriteString(w, b.String())
}

// screenSize returns the terminal size or an error naming the minimum if it is too small.
func screenSize(fd int) (rows, cols int, err error) {
	cols, rows, err = term.GetSize(fd)
	if err != nil {
		return 0, 0, err
	}
	if cols < minCols || rows < minRows {
		return 0, 0, fmt.Errorf("terminal is %dx%d, need at least %dx%d", cols, rows, minCols, minRows)
	}
	return rows, cols, nil
}

// errCrashed is game over: the stack trace is already on screen, so exit 1 without a message.
var errCrashed = errors.New("crashed")

const usage = `snake: a snake game disguised as a Spring Boot log tail.

The snake waits for your first key. Needs a terminal of at least %dx%d.

  arrows / wasd / hjkl   steer
  space                  pause; the board disappears and only log traffic shows
  q / Esc / Ctrl-C       shut the "server" down

Hitting a wall or yourself throws an exception and exits with status 1. Your score is the
request latency on the "Completed 200 OK (N ms)" line below the stack trace.

Options:
`

// tick is the game speed: one snake step per tick. Set from -t/--tick.
var tick time.Duration

func main() {
	flag.DurationVar(&tick, "tick", defaultTick, "time per snake step, e.g. 100ms or 1s")
	flag.DurationVar(&tick, "t", defaultTick, "shorthand for --tick")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, minCols, minRows)
		flag.PrintDefaults()
	}
	flag.Parse()
	if tick <= 0 {
		fmt.Fprintln(os.Stderr, "snake: --tick must be positive")
		os.Exit(2)
	}
	err := run()
	if err != nil && !errors.Is(err, errCrashed) {
		fmt.Fprintln(os.Stderr, "snake:", err)
	}
	if err != nil {
		os.Exit(1)
	}
}

// run owns the terminal; every exit path returns through its defers before main prints anything.
func run() error {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return errors.New("stdin is not a terminal")
	}
	rows, cols, err := screenSize(fd)
	if err != nil {
		return err
	}

	old, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, old)
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	keys := make(chan key)
	go readKeys(keys)

	var ring []string
	paused := false
	pending := bootLines(time.Now())
	g := newGame()
	t := time.NewTicker(bootTick)
	defer t.Stop()

loop:
	for {
		select {
		case <-sig:
			break loop
		case <-winch:
			if rows, cols, err = screenSize(fd); err != nil {
				fmt.Print("\r\n")
				return err
			}
		case k := <-keys:
			if k.quit {
				break loop
			}
			if k.pause {
				paused = !paused
				continue
			}
			g.turn(k.dir)
		case <-t.C:
			// Paused: the log takes the whole screen so the board rows are masked by traffic.
			ringCap := max(rows-boardH, 2)
			if paused {
				ringCap = rows
			}
			if len(pending) > 0 {
				ring = push(ring, ringCap, pending[0])
				pending = pending[1:]
				render(os.Stdout, ring, g, false, cols)
				if len(pending) == 0 {
					t.Reset(tick)
				}
				continue
			}
			if rand.IntN(trafficEvery) == 0 {
				ring = push(ring, ringCap, trafficLines(time.Now())...)
			}
			if paused {
				render(os.Stdout, ring, g, false, cols)
				continue
			}
			ring = push(ring, ringCap) // trims back to board height after a pause
			g.step()
			render(os.Stdout, ring, g, true, cols)
			if g.dead {
				// Board rows are stamped up to boardH*lineGap ahead of now; the trace must follow them.
				crash := crashLines(time.Now().Add(boardH*lineGap), g.cause, g.score)
				fmt.Print("\r\n" + strings.Join(crash, "\r\n") + "\r\n")
				return errCrashed
			}
		}
	}
	fmt.Print("\r\n" + strings.Join(shutdownLines(time.Now()), "\r\n") + "\r\n")
	return nil
}
