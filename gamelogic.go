package main

import (
	"errors"
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

type Ship struct {
	startx, starty, endx, endy int
}

func getEndCoords(startx, starty, boardx, boardy, length int, dir Direction, c *gin.Context) (int, int, error) {
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

	c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "something went badly wrong while building a board"})
	return 0, 0, errors.New("")
}

type Board struct {
}

func newBoard(w, h int, s ...Ship) {

}
