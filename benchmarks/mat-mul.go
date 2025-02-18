package benchmarks

import (
	"fmt"
	"math/rand/v2"
)

type product struct {
	res int
	pos position
}

type position struct {
	x int
	y int
}

func generateMatrix(rows int, cols int, valueRange int) [][]int {
	matrix := make([][]int, rows)
	for i := range matrix {
		matrix[i] = make([]int, cols)
		for j := range matrix[i] {
			matrix[i][j] = rand.IntN(valueRange)
		}
	}
	return matrix
}

func matrixMultiplication(m1 [][]int, m2 [][]int, n int) [][]int {
	if len(m1[0]) != len(m2) {
		return nil
	}
	r1 := len(m1)
	c2 := len(m2[0])

	result := make([][]int, r1)
	products := make(chan product, r1*c2)
	positions := make(chan position, r1*c2)

	for i := 0; i < n; i++ {
		go calculateProducts(m1, m2, products, positions)
	}

	for i := range result {
		result[i] = make([]int, c2)

		for j := range c2 {
			positions <- position{i, j}
		}
	}

	for i := 0; i < r1*c2; i++ {
		r := <-products
		result[r.pos.x][r.pos.y] = r.res
	}

	close(positions)
	return result
}

func calculateProduct(row []int, col []int) int {
	p := 0
	for k := 0; k < len(row); k++ {
		p += row[k] * col[k]
	}
	return p
}

func fetchColumn(m2 [][]int, j int) []int {
	var col = make([]int, len(m2))
	for i := range len(m2) {
		col[i] = m2[i][j]
	}
	return col
}

func calculateProducts(m1 [][]int, m2 [][]int, products chan product, positions chan position) {
	for pos := range positions {
		col := fetchColumn(m2, pos.y)
		products <- product{
			res: calculateProduct(m1[pos.x], col),
			pos: pos,
		}
	}
}

func RunMatrixMultiplication(n int, valueRange int, rows int, cols int) {
	m1 := generateMatrix(rows, cols, valueRange)
	m2 := generateMatrix(cols, rows, valueRange)
	fmt.Println(m1)
	fmt.Println(m2)

	res := matrixMultiplication(m1, m2, n)
	fmt.Println(res)
}
