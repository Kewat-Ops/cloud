package main

import "testing"

func TestAddition(t *testing.T) {
    if 2+2 != 4 {
        t.Errorf("Expected 4, got %d", 2+2)
    }
}
