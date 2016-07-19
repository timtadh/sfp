package subgraph

import (
	"log"
	"math"
)

import (
	"github.com/timtadh/matrix"
)


func (sg *SubGraph) Metric(o *SubGraph) float64 {
	labels := make(map[int]int, len(sg.V)+len(o.V))
	rlabels := make([]int, 0, len(sg.V)+len(o.V))
	addLabel := func(label int) {
		if _, has := labels[label]; !has {
			labels[label] = len(rlabels)
			rlabels = append(rlabels, label)
		}
	}
	for i := range sg.V {
		addLabel(sg.V[i].Color)
	}
	for i := range o.V {
		addLabel(o.V[i].Color)
	}
	for i := range sg.E {
		addLabel(sg.E[i].Color)
	}
	for i := range o.E {
		addLabel(o.E[i].Color)
	}
	W := sg.Walks(labels)
	err := W.Subtract(o.Walks(labels))
	if err != nil {
		log.Fatal(err)
	}
	W2, err := W.DenseMatrix().ElementMult(W)
	if err != nil {
		log.Fatal(err)
	}
	norm := W2.DenseMatrix().TwoNorm()
	denom := float64(len(rlabels))
	mean := norm/denom
	metric := math.Sqrt(mean)
	return metric
}

func (sg *SubGraph) LE(labels map[int]int) (L, E matrix.Matrix) {
	V := len(sg.V)
	VE := V + len(sg.E)
	L = matrix.Zeros(len(labels), VE)
	E = matrix.Zeros(VE, VE)
	for i := range sg.V {
		L.Set(labels[sg.V[i].Color], i, 1)
	}
	for i := range sg.E {
		L.Set(labels[sg.E[i].Color], V + i, 1)
	}
	for i := range sg.E {
		E.Set(sg.E[i].Src, V + i, 1)
		E.Set(V + i, sg.E[i].Targ, 1)
	}
	return L, E
}

func (sg *SubGraph) Walks(labels map[int]int) (W matrix.Matrix) {
	var err error
	L, E := sg.LE(labels)
	LT := matrix.Transpose(L)
	var En matrix.Matrix = matrix.Eye(E.Rows())
	var SEn matrix.Matrix = matrix.Zeros(E.Rows(), E.Cols())
	for i := 0; i < len(sg.V); i++ {
		En, err = En.Times(E)
		if err != nil {
			log.Fatal(err)
		}
		err = SEn.Add(En)
		if err != nil {
			log.Fatal(err)
		}
	}
	LE, err := L.Times(SEn)
	if err != nil {
		log.Fatal(err)
	}
	LELT, err := LE.Times(LT)
	if err != nil {
		log.Fatal(err)
	}
	return LELT
}
