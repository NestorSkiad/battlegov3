package main

import (
	"errors"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

// boilerplate 5ever
type Matrix[T any] struct {
	w, h int
	data []T
}

func MakeMatrix[T any](w, h int) Matrix[T] {
	return Matrix[T]{w, h, make([]T, w*h)}
}

func (m Matrix[T]) At(x, y int) T {
	return m.data[y*m.w+x]
}

func (m Matrix[T]) Set(x, y int, t T) {
	m.data[y*m.w+x] = t
}

type Direction bool

const (
	North Direction = true
	South Direction = true
	East  Direction = true
	West  Direction = true
)

var directions = []Direction{North, South, East, West}

type Ship struct {
	startx, starty, endx, endy int
	dir                        Direction
}

type Board struct {
	w, h  int
	ships []*Ship
}

func getEndCoords(startx, starty, boardx, boardy, length int, dir Direction) (int, int, error) {
	outOfBoundsError := errors.New("ship is out of bounds")

	switch dir {
	case North:
		if endy := starty + length; endy >= boardy {
			return 0, 0, outOfBoundsError
		} else {
			return startx, endy, nil
		}
	case South:
		if endy := starty - length; endy < 0 {
			return 0, 0, outOfBoundsError
		} else {
			return startx, endy, nil
		}
	case East:
		if endx := startx + length; endx >= boardx {
			return 0, 0, outOfBoundsError
		} else {
			return endx, starty, nil
		}
	case West:
		if endx := startx - length; endx < 0 {
			return 0, 0, outOfBoundsError
		} else {
			return endx, starty, nil
		}
	}

	return 0, 0, errors.New("somehow found a new direction")
}

func newBoard(w, h int, ships ...*Ship) (*Board, error) {
	outOfBoundsError := errors.New("ship is out of bounds")

	for _, ship := range ships {
		if (ship.startx >= w) || (ship.startx < 0) || (ship.starty >= h) || (ship.starty < 0) || (ship.endx >= w) || (ship.endx < 0) || (ship.endy >= h) || (ship.endy < 0) {
			return nil, outOfBoundsError
		}
	}

	return &Board{w, h, ships}, nil
}

// func addShip
func addShip(board *Board, ship *Ship) error {
	if ship.startx >= board.w || ship.endx >= board.w || ship.starty >= board.h || ship.endy >= board.h {
		return errors.New("tried to add ship into out of bounds")
	}

	board.ships = append(board.ships, ship)
	return nil
}

func shipAtCoords(board *Board, x, y int) bool {
	if (x >= board.w) || (y >= board.h) {
		return false
	}

	for _, ship := range board.ships {
		if (x >= ship.startx) && (x <= ship.endx) && (y >= ship.starty) && (y <= ship.endy) {
			return true
		}
	}
	return false
}

func newBoardFromRandom() (*Board, error) {
	dim := rand.Intn(5) + 8

	board, _ := newBoard(dim, dim)

	for i := 0; i < 3; i++ {
		startx := rand.Intn(dim)
		starty := rand.Intn(dim)

		direction := directions[rand.Intn(len(directions))]

		if endx, endy, err := getEndCoords(startx, starty, dim, dim, 3, direction); err != nil {
			i--;
			continue
		}

		// make addship function to abstract adding ships a bit, especially if I do the matrix-referencing-boats thing
	}
}
