package subgraph

import (
	"fmt"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/linked"
	"github.com/timtadh/data-structures/set"
	"github.com/timtadh/data-structures/types"
)

type Overlap struct {
	SG  *SubGraph
	Ids []*set.SortedSet // the embeddings for each vertex
}

func (sg *SubGraph) FindVertexEmbeddings(indices *Indices, minSupport int) (*Overlap, error) {
	chain := sg.edgeChain()
	b := BuildOverlap(len(sg.V), len(sg.E)).Fillable().Ctx(func(b *FillableOverlapBuilder) {
		b.SetVertex(0, sg.V[0].Color, indices.IdSet(sg.V[0].Color))
	})
	for _, e := range chain {
		errors.Logf("VE-DEBUG", "edge %v", e)
		sg.pruneVertices(minSupport, indices, b, sg.extendOverlap(indices, b, e))
		errors.Logf("VE-DEBUG", "so far %v", b)
	}
	return nil, errors.Errorf("unfinished")
}

func (sg *SubGraph) extendOverlap(indices *Indices, b *FillableOverlapBuilder, e *Edge) (dirty *linked.UniqueDeque) {
	src := b.V[e.Src].Idx
	targ := b.V[e.Targ].Idx

	if src == -1 && targ == -1 {
		panic("src and targ == -1. Which means the edge chain was not connected.")
	} else if src != -1 && targ != -1 {
		b.AddEdge(&b.V[e.Src], &b.V[e.Targ], e.Color)
	} else if src != -1 {
		targs := set.NewSortedSet(10)
		for srcId, next := b.Ids[src].Items()(); next != nil; srcId, next = next() {
			for _, targ := range indices.TargsFromSrc(int(srcId.(types.Int)), e.Color, sg.V[e.Targ].Color, nil) {
				targs.Add(types.Int(targ))
			}
		}
		b.SetVertex(e.Targ, sg.V[e.Targ].Color, targs)
		b.AddEdge(&b.V[e.Src], &b.V[e.Targ], e.Color)
	} else if targ != -1 {
		srcs := set.NewSortedSet(10)
		for targId, next := b.Ids[targ].Items()(); next != nil; targId, next = next() {
			for _, targ := range indices.SrcsToTarg(int(targId.(types.Int)), e.Color, sg.V[e.Src].Color, nil) {
				srcs.Add(types.Int(targ))
			}
		}
		b.SetVertex(e.Src, sg.V[e.Src].Color, srcs)
		b.AddEdge(&b.V[e.Src], &b.V[e.Targ], e.Color)
	} else {
		panic("unreachable")
	}
	dirty = linked.NewUniqueDeque()
	dirty.Push(types.Int(e.Src))
	dirty.Push(types.Int(e.Targ))
	return dirty
}

func (sg *SubGraph) pruneVertices(minSupport int, indices *Indices, b *FillableOverlapBuilder, dirty *linked.UniqueDeque) (unsup bool) {
	for dirty.Size() > 0 {
		idx, err := dirty.DequeBack()
		if err != nil {
			panic(errors.Errorf("should not be possible").(*errors.Error).Chain(err))
		}
		unsup = sg.pruneVertex(int(idx.(types.Int)), minSupport, indices, b, dirty)
		if unsup {
			return true
		}
	}
	return false
}

func (sg *SubGraph) pruneVertex(idx, minSupport int, indices *Indices, b *FillableOverlapBuilder, dirty *linked.UniqueDeque) (unsup bool) {
	// todo next
	if b.Ids[idx].Size() < minSupport {
		return true
	}
	changed := false
	for id, next := b.Ids[idx].Copy().Items()(); next != nil; id, next = next() {
		if !sg.hasEveryEdge(idx, int(id.(types.Int)), indices, b) {
			b.Ids[idx].Delete(id)
			changed = true
		}
		if b.Ids[idx].Size() < minSupport {
			return true
		}
	}
	if changed {
		for _, eidx := range b.Adj[idx] {
			e := &b.E[eidx]
			if e.Src != idx {
				dirty.EnqueFront(types.Int(e.Src))
			}
			if e.Targ != idx {
				dirty.EnqueFront(types.Int(e.Targ))
			}
		}
	}
	return false
}

func (sg *SubGraph) hasEveryEdge(idx, id int, indices *Indices, b *FillableOverlapBuilder) (bool) {
	for _, eidx := range b.Adj[idx] {
		e := &b.E[eidx]
		found := false
		if e.Src == idx {
			srcId := id
			for tid, next := b.Ids[e.Targ].Items()(); next != nil; tid, next = next() {
				targId := int(tid.(types.Int))
				if indices.HasEdge(srcId, targId, e.Color) {
					found = true
					break
				}
			}
		} else {
			targId := id
			for sid, next := b.Ids[e.Src].Items()(); next != nil; sid, next = next() {
				srcId := int(sid.(types.Int))
				if indices.HasEdge(srcId, targId, e.Color) {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (o *Overlap) String() string {
	V := make([]string, 0, len(o.SG.V))
	E := make([]string, 0, len(o.SG.E))
	for i := range o.SG.V {
		V = append(V, fmt.Sprintf(
			"(%v:%v)",
			o.SG.V[i].Color,
			o.Ids[i],
		))
	}
	for _, e := range o.SG.E {
		E = append(E, fmt.Sprintf(
			"[%v->%v:%v]",
			e.Src,
			e.Targ,
			e.Color,
		))
	}
	return fmt.Sprintf("{%v:%v}%v%v", len(o.SG.E), len(o.SG.V), strings.Join(V, ""), strings.Join(E, ""))
}