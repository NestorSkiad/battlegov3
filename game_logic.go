package main

import (
	"errors"
	"math/rand"
)

// possible use: secondary index for Boat pointers (won't have to linear check for boats given coords)
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

// type system shenanigans
type Direction int

const (
	North Direction = iota
	East
	South
	West
)

type Player int

const (
	Host Player = iota
	Guest
)

var directions = []Direction{North, South, East, West}

type Ship struct {
	startx, starty, endx, endy int
	dir                        Direction
	alive					   bool
}

type Board struct {
	w, h  int
	ships []*Ship
}

type Move struct {
	x, y int
	hit bool
}

// TODO: newGameState func
type GameState struct {
	boardHost *Board
	boardGuest *Board
	evens Player
	moves []*Move
}

func (m *GameState) makeMove() error {
	return nil
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

func newShip(startx, starty, endx, endy int, direction Direction) *Ship {
	return &Ship{startx, starty, endx, endy, direction, true}
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
func (board *Board) addShip(ship *Ship) error {
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

		// maybe randomise ship length
		endx, endy, err := getEndCoords(startx, starty, board.w, board.h, 3, direction)
		if err != nil {
			i--
			continue
		}

		ship := newShip(startx, starty, endx, endy, direction)
		board.addShip(ship)
		// make addship function to abstract adding ships a bit, especially if I do the matrix-referencing-boats thing
	}

	return board, nil;
}
