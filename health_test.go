package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToMegaBytes(t *testing.T) {
	cases := []struct {
		value    uint64
		expected float64
	}{
		{1024, 0},
		{1024 * 1024, 1},
		{1024 * 1024 * 10, 10},
		{1024 * 1024 * 100, 100},
		{1024 * 1024 * 250, 250},
	}

	for _, c := range cases {
		actual := toMegaBytes(c.value)
		assert.Equal(t, c.expected, actual)
	}
}

func TestRound(t *testing.T) {
	cases := []struct {
		value    float64
		expected int
	}{
		{0, 0},
		{1, 1},
		{1.56, 2},
		{1.38, 1},
		{30.12, 30},
	}

	for _, c := range cases {
		actual := round(c.value)
		assert.Equal(t, c.expected, actual)
	}
}

func TestToFixed(t *testing.T) {
	cases := []struct {
		value    float64
		expected float64
	}{
		{0, 0},
		{1, 1},
		{123, 123},
		{0.99, 1},
		{1.02, 1},
		{1.82, 1.8},
		{1.56, 1.6},
		{1.38, 1.4},
	}

	for _, c := range cases {
		actual := toFixed(c.value, 1)
		assert.Equal(t, c.expected, actual)
	}
}
