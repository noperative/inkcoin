package main

import (
	"fmt"

	"../shapelib"
)

func main() {
	// RANDOM SHAPE
	points := make([]shapelib.Point, 10)
	points[0] = shapelib.Point{0, 60, false}
	points[1] = shapelib.Point{150, 90, false}
	points[2] = shapelib.Point{160, 299, false}
	points[3] = shapelib.Point{450, 150, false}
	points[4] = shapelib.Point{599, 250, false}
	points[5] = shapelib.Point{590, 30, false}
	points[6] = shapelib.Point{299, 150, false}
	points[7] = shapelib.Point{150, 30, false}
	points[8] = shapelib.Point{0, 30, false}
	points[9] = shapelib.Point{0, 60, false}

	// HEXAGON
	//	points := make([]shapelib.Point, 7 )
	//	points[0] = shapelib.Point {350, 299, false }
	//	points[1] = shapelib.Point {480, 160, false }
	//	points[2] = shapelib.Point {350, 0, false }
	//	points[3] = shapelib.Point {140, 0, false }
	//	points[4] = shapelib.Point {10, 160, false }
	//	points[5] = shapelib.Point {140, 299, false }
	//	points[6] = shapelib.Point {350, 299, false }

	// HEXAGON ROTATED
	//	points := make([]shapelib.Point, 7 )
	//	points[0] = shapelib.Point {140, 299, false }
	//	points[1] = shapelib.Point {350, 299, false }
	//	points[2] = shapelib.Point {480, 160, false }
	//	points[3] = shapelib.Point {350, 0, false }
	//	points[4] = shapelib.Point {140, 0, false }
	//	points[5] = shapelib.Point {10, 160, false }
	//	points[6] = shapelib.Point {140, 299, false }

	// MOVE DOUBLE RECTANGLE
	//	points := make([]shapelib.Point, 10)
	//	points[0] = shapelib.Point {600, 200, false }
	//	points[1] = shapelib.Point {600, 0, false }
	//	points[2] = shapelib.Point {0, 0, false }
	//	points[3] = shapelib.Point {0, 200, false }
	//	points[4] = shapelib.Point {600, 200, false }
	//
	//	points[5] = shapelib.Point {550, 175, true }
	//	points[6] = shapelib.Point {50, 175, false }
	//	points[7] = shapelib.Point {50, 25, false }
	//	points[8] = shapelib.Point {550, 25, false }
	//	points[9] = shapelib.Point {550, 175, false }

	// SQUARE
	//points := make([]shapelib.Point, 5)
	//points[0] = shapelib.Point {0, 0, false }
	//points[1] = shapelib.Point {0, 300, false }
	//points[2] = shapelib.Point {300, 300, false }
	//points[3] = shapelib.Point {300, 0, false }
	//points[4] = shapelib.Point {0, 0, false }

	path1 := shapelib.NewPath(points, true, true)
	sub1 := path1.SubArray()
	fmt.Println("Total len path:", path1.TotalLength())

	// CIRCLE
	//	circ := shapelib.NewCircle(150, 150, 100, true )
	//	sub2 := circ.SubArray()
	//	fmt.Println("Circumference:", circ.Circumference())

	fmt.Println("Pixels filled:", sub1.PixelsFilled())
	_, cost := path1.SubArrayAndCost()
	fmt.Println("Cost:", cost)

	a := shapelib.NewPixelArray(600, 400)
	a.MergeSubArray(sub1)
	//fmt.Println("Square circle conflict?", a.HasConflict(sub2))
}
