package main

import "testing"

// straight builds a horizontal snake of n segments ending at head, moving right.
func straight(n int, head point) *game {
	g := &game{dir: right, next: right, food: point{-1, -1}}
	for i := n - 1; i >= 0; i-- {
		g.snake = append(g.snake, point{head.x - i, head.y})
	}
	return g
}

func TestStepMoves(t *testing.T) {
	g := straight(3, point{5, 5})
	g.step()
	if got := g.head(); got != (point{6, 5}) {
		t.Fatalf("head = %v, want {6 5}", got)
	}
	if len(g.snake) != 3 || g.dead {
		t.Fatalf("len = %d dead = %v", len(g.snake), g.dead)
	}
}

func TestGrowOnFood(t *testing.T) {
	g := straight(3, point{5, 5})
	g.food = point{6, 5}
	g.step()
	if len(g.snake) != 4 || g.score != 1 {
		t.Fatalf("len = %d score = %d", len(g.snake), g.score)
	}
	if g.food == (point{6, 5}) || g.occupied(g.food) {
		t.Fatalf("food not relocated: %v", g.food)
	}
}

func TestWallDeath(t *testing.T) {
	g := straight(3, point{boardW - 2, 5})
	g.step()
	if g.dead {
		t.Fatal("died one cell early")
	}
	g.step()
	if !g.dead || g.cause != causeWall {
		t.Fatalf("dead = %v cause = %q", g.dead, g.cause)
	}
}

func TestSelfCollision(t *testing.T) {
	// Hook shape: head {5,6} moving up into {5,5}, which is body, not the tail.
	g := &game{dir: up, next: up, food: point{-1, -1}}
	g.snake = []point{{4, 5}, {5, 5}, {6, 5}, {6, 6}, {5, 6}}
	g.step()
	if !g.dead || g.cause != causeSelf {
		t.Fatalf("dead = %v cause = %q", g.dead, g.cause)
	}
}

func TestTailChase(t *testing.T) {
	// 2x2 loop: tail {5,5}, head {5,6} moving up into the tail cell, which vacates this step.
	g := &game{dir: up, next: up, food: point{-1, -1}}
	g.snake = []point{{5, 5}, {6, 5}, {6, 6}, {5, 6}}
	g.step()
	if g.dead {
		t.Fatalf("tail chase should be legal, cause = %q", g.cause)
	}
}

func TestNoReverse(t *testing.T) {
	g := straight(3, point{5, 5})
	g.turn(left)
	if g.next != right {
		t.Fatalf("reversed: next = %v", g.next)
	}
	g.turn(up)
	g.turn(left) // same tick: dir is still right, so left must still be refused
	if g.next != up {
		t.Fatalf("two-key reverse: next = %v", g.next)
	}
}

func TestWaitsForFirstKey(t *testing.T) {
	g := newGame()
	before := g.head()
	g.step()
	if g.head() != before {
		t.Fatal("moved before any key was pressed")
	}
	g.turn(up)
	g.step()
	if g.head() == before {
		t.Fatal("did not move after key")
	}
}
