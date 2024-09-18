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

// Direction of where the ship points
type Direction int

// FIXME: this won't work. convert to horizontal/vertical
// Ship directionality values
const (
	Horizontal Direction = iota
	Vertical
)

var directions = []Direction{Horizontal, Vertical}

// PlayerType demarkates either host or guest
type PlayerType int

// PlayerType values
const (
	Host PlayerType = iota
	Guest
)

var players = []PlayerType{Host, Guest}

// TODO: json tags
// A ship in Battleship
type Ship struct {
	startx int
	starty int
	endx int
	endy int
	dir Direction
	alive bool
}

// Board abstraction, with dimensions and ships
type Board struct {
	W int `json:"width"`
	H  int `json:"height"`
	Ships []*Ship `json:"ships"`
}

// Move made by a player
type Move struct {
	X int `json:"x"`
	Y int `json:"y"`
	Hit bool `json:"hit"`
}

// GameState represents a game
type GameState struct {
	boardHost *Board
	boardGuest *Board
	evens PlayerType
	moves []*Move
}

// implement presentablegamestate struct, as in https://github.com/gin-gonic/gin/issues/715#issuecomment-381302094
// implement GS.toPresentable which takes Player and returns Presentable object with only that player's board
// add json bindings to ship, board, move

func newGameState() (*GameState, error) {
	gs := &GameState{}

	dim := rand.Intn(5) + 8

	boardHost, err := newBoardFromRandom(dim)
	if err != nil {
		return nil, err
	}
	gs.boardHost = boardHost

	boardGuest, err := newBoardFromRandom(dim)
	if err != nil {
		return nil, err
	}
	gs.boardGuest = boardGuest

	gs.evens = players[rand.Intn(len(players))]

	gs.moves = []*Move{}
	
	return gs, nil
}

func (m *GameState) makeMove() error {
	return nil
}

func getEndCoords(startx, starty, boardx, boardy, length int, dir Direction) (int, int, error) {
	outOfBoundsError := errors.New("ship is out of bounds")

	switch dir {
	case Vertical:
		if endy := starty + length; endy >= boardy {
			return 0, 0, outOfBoundsError
		} else {
			return startx, endy, nil
		}
	case Horizontal:
		if endx := startx + length; endx >= boardx {
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
	if ship.startx >= board.W || ship.endx >= board.W || ship.starty >= board.H || ship.endy >= board.H {
		return errors.New("tried to add ship into out of bounds")
	}

	board.Ships = append(board.Ships, ship)
	return nil
}

func (board *Board) shipAtCoords(x, y int) bool {
	if (x >= board.W) || (y >= board.H) {
		return false
	}

	for _, ship := range board.Ships {
		if (x >= ship.startx) && (x <= ship.endx) && (y >= ship.starty) && (y <= ship.endy) {
			return true
		}
	}
	return false
}

func newBoardFromRandom(dim int) (*Board, error) {
	board, _ := newBoard(dim, dim)

	for i := 0; i < 3; i++ {
		startx := rand.Intn(dim)
		starty := rand.Intn(dim)

		direction := directions[rand.Intn(len(directions))]

		// maybe randomise ship length
		endx, endy, err := getEndCoords(startx, starty, board.W, board.H, 3, direction)
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
