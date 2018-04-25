/*

This package is intended to be used to verify that shapes are not conflicting
with each other.


Public functions:

	NewPath(points []Point, filled bool, strokeTransparent bool) -> Path

	NewCircle(xc, yc, radius int, filled bool, strokeTransparent bool) -> Circle

	NewPixelArray(xMax int, yMax int) -> PixelArray

	NewPixelSubArray(xStart, xEnd, yStart, yEnd int) -> PixelSubArray


Public types and methods:

	PixelArray
	  Print()
	  HasConflict(sub PixelSubArray) -> bool
	  MergeSubArray(sub PixelSubArray)

	PixelSubArray
	  Print()
	  PixelsFilled() -> int

	Point

	Shape
	  SubArray()        -> PixelSubArray
	  SubArrayAndCost() -> int

	Path
	  SubArray()        -> PixelSubArray
	  TotalLength()     -> int
	  SubArrayAndCost() -> int

	Circle
	  SubArray()        -> PixelSubArray
	  Circumference()   -> int
	  SubArrayAndCost() -> int


This file in particular contains all type definitions and some misc. functions.

*/

package shapelib

/*******************
* TYPE_DEFINITIONS *
*******************/

// Array of pixels. First index represents y, 2nd index represent x.
// Byte array is compressed so one bit represents one pixel.
type PixelArray [][]byte

// SubArray that starts at a relative position rather than (0,0)
// xStartByte should be on a byte boundary, ie. % 8 == 0.
type PixelSubArray struct {
	bytes      [][]byte
	xStartByte int
	yStart     int
}

// Interface for a shape that can return its subarray of pixel filled.
type Shape interface {

	// Returns the PixelSubArray that represents the pixels filled on
	// a pixel array for this particular shape.
	SubArray() PixelSubArray

	// Returns the PixelSubArray that represents the pixels filled on
	// a pixel array for this particular shape, as well as the cost that
	// is associated with the shape.
	SubArrayAndCost() (subarr PixelSubArray, cost int)
}

// Represents the data of a Path SVG item.
// Any closed shape must have the last point in the
// array be equal to the first point. So a quadrilateral
// should have len(Points) == 5 and Points[0] is equal
// to Points[4].
type Path struct {
	Points            []Point
	Filled            bool
	StrokeFilled bool

	// The below 4 values should create a rectangle that
	// can fit the entire path within it.
	XMin              int
	XMax              int
	YMin              int
	YMax              int
}

// Point. Represents a point or pixel on a discrete 2D array.
//
// All points should be in the 1st quadrant (x >= 0, y >= 0).
//
// Moved is only relevant for the Path object. Its relevance for Path is thus:
// if the Point (n) has (Moved == true), then there is no line drawn between
// Point (n-1) to Point(n).
type Point struct {
	X     int
	Y     int
	Moved bool
}

// Circle. Not much more to say really.
type Circle struct {
	C                 Point
	R                 int
	Filled            bool
	StrokeFilled bool
}

// Used for computing shit for the Path object.
type slopeType int

const (
	POSRIGHT slopeType = iota
	NEGRIGHT
	POSLEFT
	NEGLEFT
	INFUP
	INFDOWN
)

/***********************
* FUNCTION_DEFINITIONS *
************************/

// Gets the maximum byte index for a PixelArray's columns.
// Computes the minimum number of bytes needed to contain
// nBits bits.
func maxByte(nBits int) int {
	rem := 0
	if (nBits % 8) != 0 {
		rem = 1
	}

	return (nBits / 8) + rem
}

// Get the slope and y-intercept of a line formed by two points
func getSlopeIntercept(p1 Point, p2 Point) (slope float64, intercept float64) {
	slope = (float64(p2.Y) - float64(p1.Y)) / (float64(p2.X) - float64(p1.X))
	intercept = float64(p1.Y) - slope*float64(p1.X)

	return slope, intercept
}

// Get slope type, slope, and intercept for a a pair of points
func getLineParams(p1, p2 Point) (sT slopeType, slope, intercept float64) {
	if p1.X == p2.X {
		// Check for infinite slope.
		if p2.Y > p1.Y {
			//fmt.Println("INFUP slope")
			sT = INFUP
		} else {
			//fmt.Println("INFDOWN slope")
			sT = INFDOWN
		}

		slope, intercept = 0, 0
	} else {
		// 4 classifications of non infinite slope based
		// on the relative positions of p1 and p2
		slope, intercept = getSlopeIntercept(p1, p2)
		if p1.X < p2.X {
			if slope >= 0 {
				//fmt.Println("POSRIGHT slope")
				sT = POSRIGHT
			} else {
				//fmt.Println("NEGRIGHT slope")
				sT = NEGRIGHT
			}
		} else {
			if slope >= 0 {
				//fmt.Println("POSLEFT slope")
				sT = POSLEFT
			} else {
				//fmt.Println("NEGLEFT slope")
				sT = NEGLEFT
			}
		}
	}

	return sT, slope, intercept
}

// Generates an iterator for a line.  What a mess.
func linePointsGen(p1, p2 Point) (gen func() (x, y int), vertDirection int) {
	// Set up math
	slopeT, slope, intercept := getLineParams(p1, p2)

	x := float64(p1.X)
	xPrev := int(x)
	y := p1.Y
	yThresh := 0

	// Every slope type has a different iterator, since they change the
	// x and y values in different combinations, as well as do different
	// comparisons on the values.
	switch slopeT {
	case POSRIGHT:
		if slope == 0 {
			vertDirection = 0
		} else {
			vertDirection = 1
		}

		return func() (int, int) {
			if y < yThresh {
				if y > p2.Y {
					return -1, -1
				}

				y++
				return xPrev, y
			} else {
				if int(x) > p2.X {
					return -1, -1
				}

				yThresh = int(slope*x + intercept + 0.5)
				xPrev = int(x)
				x++

				if y != yThresh {
					y++
				}

				return xPrev, y
			}
		}, vertDirection
	case NEGRIGHT:
		vertDirection = -1
		yThresh = int(slope*x + intercept + 0.5)

		return func() (int, int) {
			if y > yThresh {
				if y < p2.Y {
					return -1, -1
				}

				y--
				return xPrev, y
			} else {
				if int(x) > p2.X {
					return -1, -1
				}

				yThresh = int(slope*x + intercept + 0.5)
				xPrev = int(x)
				x++

				if y != yThresh {
					y--
				}

				return xPrev, y
			}
		}, vertDirection
	case POSLEFT:
		if slope == 0 {
			vertDirection = 0
		} else {
			vertDirection = -1
		}

		yThresh = int(slope*x + intercept + 0.5)

		return func() (int, int) {
			if y > yThresh {
				if y < p2.Y {
					return -1, -1
				}

				y--
				return xPrev, y
			} else {
				if int(x) < p2.X {
					return -1, -1
				}

				yThresh = int(slope*x + intercept + 0.5)
				xPrev = int(x)
				x--

				if y != yThresh {
					y--
				}

				return xPrev, y
			}
		}, vertDirection
	case NEGLEFT:
		vertDirection = 1

		return func() (int, int) {
			if y < yThresh {
				if y > p2.Y {
					return -1, -1
				}

				y++
				return xPrev, y
			} else {
				if int(x) < p2.X {
					return -1, -1
				}

				yThresh = int(slope*x + intercept + 0.5)
				xPrev = int(x)
				x--

				if y != yThresh {
					y++
				}

				return xPrev, y
			}
		}, vertDirection
	case INFUP:
		vertDirection = 1

		return func() (int, int) {
			if y > p2.Y {
				return -1, -1
			}

			yPrev := y
			y++
			return int(x), yPrev
		}, vertDirection
	case INFDOWN:
		vertDirection = -1

		return func() (int, int) {
			if y < p2.Y {
				return -1, -1
			}

			yPrev := y
			y--
			return int(x), yPrev
		}, vertDirection
	}

	return nil, -1
}
