/*

This file contains functions related to PixelArray and PixelSubArray.

*/
package shapelib

import (
	"fmt"
	"math/bits"
)

/************************
* PIXEL_ARRAY_FUNCTIONS *
************************/

// Returns a new pixel array that is fully zeroed.
func NewPixelArray(xMax int, yMax int) PixelArray {
	// create rows
	a := make([][]byte, yMax + 1)

	// Initialize the number of bytes required in each row
	xSz := maxByte(xMax + 1)

	for y := 0; y < yMax + 1; y++ {
		// create compressed columns (one bit per pixel)
		a[y] = make([]byte, xSz)

		// zero fill
		for x := 0; x < xSz; x++ {
			a[y][x] = 0
		}
	}

	return a
}

// Checks if there is a conflict between the PixelArray and a PixelSubArray.
func (a PixelArray) HasConflict(sub PixelSubArray) bool {
	xLastByte := sub.xStartByte + len(sub.bytes[0])
	yLast := sub.yStart + len(sub.bytes)

	// Do some basic validations for overflow
	if xLastByte > len(a[0]) {
		fmt.Println("Sub array is past the x boundary")
		return true
	}

	if (sub.yStart + len(sub.bytes)) > len(a) {
		fmt.Println("Sub array is past the y boundary:")
		fmt.Println("yMaxSub, yMax:", sub.yStart+len(sub.bytes[0]), len(a))
		return true
	}

	// Compare the bytes using bitwise &. If there is a conflict,
	// there should be some bitwise & that != 0.
	for y := sub.yStart; y < yLast; y++ {
		ySub := y - sub.yStart

		for x := sub.xStartByte; x < xLastByte; x++ {
			xSub := x - sub.xStartByte

			if (a[y][x] & sub.bytes[ySub][xSub]) != 0 {
				fmt.Println("Conflict at (x y):", x, y)
				return true
			}
		}
	}

	return false
}

// Applies all of the filled bits in the sub-array to the pixel array
func (a *PixelArray) MergeSubArray(sub PixelSubArray) {
	yLast := sub.yStart + len(sub.bytes)

	// Do some basic validations for overflow
	if (sub.xStartByte + len(sub.bytes[0])) > len((*a)[0]) {
		fmt.Println("Sub array is past the x boundary")
		return
	}

	if (sub.yStart + len(sub.bytes)) > len(*a) {
		fmt.Println("Sub array is past the y boundary:")
		fmt.Println("yMaxSub, yMax:", sub.yStart+len(sub.bytes), len(*a))
		fmt.Println("Sub array is past the y boundary")
		return
	}

	xLastByte := sub.xStartByte + len(sub.bytes[0])

	for y := sub.yStart; y < yLast; y++ {
		ySub := y - sub.yStart

		for x := sub.xStartByte; x < xLastByte; x++ {
			xSub := x - sub.xStartByte

			(*a)[y][x] |= sub.bytes[ySub][xSub]
		}
	}
}

// Prints the bits in the array.
func (a PixelArray) Print() {
	for y := len(a) - 1; y >= 0; y-- {
		fmt.Printf("%d\t", y)
		for x := 0; x < len(a[0]); x++ {
			fmt.Printf("%b%b%b%b%b%b%b%b",
				(a[y][x])&1,
				(a[y][x]>>1)&1,
				(a[y][x]>>2)&1,
				(a[y][x]>>3)&1,
				(a[y][x]>>4)&1,
				(a[y][x]>>5)&1,
				(a[y][x]>>6)&1,
				(a[y][x]>>7)&1)
		}

		fmt.Printf("\n")
	}
}

/****************************
* PIXEL_SUB_ARRAY_FUNCTIONS *
****************************/

// Returns a new pixel sub array.
func NewPixelSubArray(xStart int, xEnd int, yStart int, yEnd int) PixelSubArray {
	// Set up the values for the sub array struct
	xStartByte := xStart / 8
	xSizeByte := maxByte(xEnd+1) - xStartByte
	ySize := yEnd - yStart + 1

	a := make([][]byte, ySize)

	for y := 0; y < ySize; y++ {
		a[y] = make([]byte, xSizeByte)

		for x := 0; x < xSizeByte; x++ {
			a[y][x] = 0
		}
	}

	return PixelSubArray{a, xStartByte, yStart}
}

// Set the bit on the given co-ordinate
func (a *PixelSubArray) set(x, y int) {
	xByte := x/8 - a.xStartByte
	xBit := uint(x % 8)
	yRow := y - a.yStart

	if yRow > len(a.bytes) {
		fmt.Println("Y OOB:", len(a.bytes), yRow)
	}

	if xByte > len(a.bytes[0]) {
		fmt.Println("X OOB:", len(a.bytes[0]), xByte)
	}

	a.bytes[yRow][xByte] |= (1 << xBit)
}

// Flip all of the bits in the sub array to the right of the provided coordinate
func (a *PixelSubArray) flipAllRight(x, y int) {
	xBit := uint(x % 8)
	xByte := x/8 - a.xStartByte
	yRow := y - a.yStart

	for i := xBit; i < 8; i++ {
		a.bytes[yRow][xByte] ^= (1 << i)
	}

	for i := xByte + 1; i < len(a.bytes[0]); i++ {
		a.bytes[yRow][i] ^= 0xFF
	}
}

// Fill in between the two coordinates formed by (xl,y) and (xr,y)
func (a *PixelSubArray) fillBetween(xl, xr, y int) {
	yRow := y - a.yStart
	xByteL := xl/8 - a.xStartByte
	xByteR := xr/8 - a.xStartByte

	// Fill in the partial bits on left
	xBit := uint8(xl % 8)
	for i := xBit; i < 8; i++ {
		a.bytes[yRow][xByteL] |= (1 << i)
	}

	// Fill in the partial bits on right
	xBit = uint8(xr % 8)
	for i := xBit; i < 255; i-- {
		a.bytes[yRow][xByteR] |= (1 << i)
	}

	// Fill in bytes in between
	for i := xByteL + 1; i < xByteR; i++ {
		a.bytes[yRow][i] |= 0xFF
	}
}

// Prints the bits in the array. There is no on the screen
// for where the sub-array is meant to be located
func (a PixelSubArray) Print() {
	for y := len(a.bytes) - 1; y >= 0; y-- {
		for x := 0; x < len(a.bytes[0]); x++ {
			fmt.Printf("%b%b%b%b%b%b%b%b",
				(a.bytes[y][x])&1,
				(a.bytes[y][x]>>1)&1,
				(a.bytes[y][x]>>2)&1,
				(a.bytes[y][x]>>3)&1,
				(a.bytes[y][x]>>4)&1,
				(a.bytes[y][x]>>5)&1,
				(a.bytes[y][x]>>6)&1,
				(a.bytes[y][x]>>7)&1)
		}

		fmt.Printf("\n")
	}
}

// Get the number of pixels filled in the sub array
func (a PixelSubArray) PixelsFilled() int {
	sum := int(0)
	for y := len(a.bytes) - 1; y >= 0; y-- {
		for x := 0; x < len(a.bytes[0]); x++ {
			sum += bits.OnesCount8(a.bytes[y][x])
		}
	}

	return sum
}
