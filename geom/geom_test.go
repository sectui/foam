package geom

import (
	"slices"
	"testing"
)

func TestDistribute(t *testing.T) {
	got := Distribute(100, []Size{Fixed(20), Ratio(1), Ratio(2)})
	if want := []int{20, 26, 54}; !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := Distribute(10, []Size{Fixed(30), {}}); !slices.Equal(got, []int{10, 0}) {
		t.Fatalf("overflow: got %v", got)
	}
}

func TestNeighbor(t *testing.T) {
	left := Rect{0, 0, 20, 30}
	top := Rect{20, 0, 80, 15}
	bottom := Rect{20, 15, 80, 15}
	rs := []Rect{left, top, bottom}
	if i := Neighbor(left, rs[1:], Right); i != 0 {
		t.Errorf("right of left = %d", i)
	}
	if i := Neighbor(top, rs, Down); i != 2 {
		t.Errorf("below top = %d", i)
	}
	if i := Neighbor(bottom, rs, Left); i != 0 {
		t.Errorf("left of bottom = %d", i)
	}
	if i := Neighbor(left, rs, Left); i != -1 {
		t.Errorf("left of left = %d", i)
	}
}

func TestColumns(t *testing.T) {
	got := Rect{X: 2, Y: 1, W: 10, H: 3}.Columns(Fixed(4), Ratio(1))
	if got[0] != (Rect{2, 1, 4, 3}) || got[1] != (Rect{6, 1, 6, 3}) {
		t.Fatalf("got %v", got)
	}
}
