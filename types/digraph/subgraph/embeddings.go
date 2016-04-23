package subgraph

import ()

import (
	"github.com/timtadh/data-structures/hashtable"
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/types"
	"github.com/timtadh/data-structures/set"
	"github.com/timtadh/goiso"
)

import (
	"github.com/timtadh/sfp/stores/int_int"
)

// Tim You Are Here:
// You just ran %s/\*goiso.SubGraph/*Embedding/g
//
// Now it is time to transition this thing over to *Embeddings
// Next it is time to create stores for *Embeddings
// Then it is time to transition types/digraph to *Embeddings


type EmbIterator func() (*Embedding, EmbIterator)
type Pruner func(leastCommonVertex int, chain []*Edge) func(emb *FillableEmbeddingBuilder) bool

func (sg *SubGraph) Embeddings(G *goiso.Graph, ColorMap int_int.MultiMap) ([]*Embedding, error) {
	embeddings := make([]*Embedding, 0, 10)
	err := sg.DoEmbeddings(G, ColorMap, func(emb *Embedding) error {
		embeddings = append(embeddings, emb)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return embeddings, nil

}

func (sg *SubGraph) DoEmbeddings(G *goiso.Graph, ColorMap int_int.MultiMap, do func(*Embedding) error) error {
	ei, err := sg.IterEmbeddings(G, ColorMap, nil)
	if err != nil {
		return err
	}
	for emb, next := ei(); next != nil; emb, next = next() {
		err := do(emb)
		if err != nil {
			return err
		}
	}
	return nil
}

func FilterAutomorphs(it EmbIterator, err error) (ei EmbIterator, _ error) {
	if err != nil {
		return nil, err
	}
	idSet := func(emb *Embedding) *set.SortedSet {
		ids := set.NewSortedSet(len(emb.Ids))
		for id := range emb.Ids {
			ids.Add(types.Int(id))
		}
		return ids
	}
	seen := hashtable.NewLinearHash()
	ei = func() (emb *Embedding, _ EmbIterator) {
		if it == nil {
			return nil, nil
		}
		for emb, it = it(); it != nil; emb, it = it() {
			ids := idSet(emb)
			if !seen.Has(ids) {
				seen.Put(ids, nil)
				return emb, ei
			}
		}
		return nil, nil
	}
	return ei, nil
}

func (sg *SubGraph) IterEmbeddings(G *goiso.Graph, ColorMap int_int.MultiMap, pruner Pruner) (ei EmbIterator, err error) {
	type entry struct {
		emb *FillableEmbeddingBuilder
		eid int
	}
	pop := func(stack []entry) (entry, []entry) {
		return stack[len(stack)-1], stack[0 : len(stack)-1]
	}

	if len(sg.V) == 0 {
		ei = func() (*Embedding, EmbIterator) {
			return nil, nil
		}
		return ei, nil
	}
	chain := sg.EdgeChain()
	vembs, err := sg.StartEmbeddings(G, ColorMap)
	if err != nil {
		return nil, err
	}

	var prune func(*FillableEmbeddingBuilder) bool = nil
	if pruner != nil {
		prune = pruner(0, chain)
	}

	// seen := hashtable.NewLinearHash()
	stack := make([]entry, 0, len(vembs)*2)
	for _, vemb := range vembs {
		stack = append(stack, entry{vemb, 0})
	}

	ei = func() (*Embedding, EmbIterator) {
		for len(stack) > 0 {
			var i entry
			i, stack = pop(stack)
			if prune != nil && prune(i.emb) {
				continue
			}
			// otherwise success we have an embedding we haven't seen
			if i.eid >= len(chain) {
				// check that this is the subgraph we sought
				emb := i.emb.Build()
				errors.Logf("FOUND", "\n  builder %v %v\n    built %v\n  pattern %v", i.emb.Builder, i.emb.Ids, emb, emb.SG)
				if sg.Matches(emb) {
					// sweet we can yield this embedding!
					return emb, ei
				}
				// nope wasn't an embedding drop it
			} else {
				// ok extend the embedding
				errors.Logf("DEBUG", "\n  extend %v %v %v", i.emb.Builder, i.emb.Ids, chain[i.eid])
				for _, ext := range sg.ExtendEmbedding(G, i.emb, chain[i.eid]) {
					/*label := types.ByteSlice(ext.Serialize())
					if seen.Has(label) {
						continue
					}
					seen.Put(label, nil)*/
					stack = append(stack, entry{ext, i.eid + 1})
				}
				errors.Logf("DEBUG", "stack len %v", len(stack))
			}
		}
		return nil, nil
	}
	return ei, nil
}

func (sg *SubGraph) Matches(emb *Embedding) bool {
	if len(sg.V) != len(emb.SG.V) {
		return false
	}
	if len(sg.E) != len(emb.SG.E) {
		return false
	}
	for i := range sg.V {
		if sg.V[i].Color != emb.SG.V[i].Color {
			return false
		}
	}
	for i := range sg.E {
		if sg.E[i].Src != emb.SG.E[i].Src {
			return false
		}
		if sg.E[i].Targ != emb.SG.E[i].Targ {
			return false
		}
		if sg.E[i].Color != emb.SG.E[i].Color {
			return false
		}
	}
	return true
}

func (sg *SubGraph) StartEmbeddings(G *goiso.Graph, ColorMap int_int.MultiMap) ([]*FillableEmbeddingBuilder, error) {
	color := sg.V[0].Color
	embs := make([]*FillableEmbeddingBuilder, 0, G.ColorFrequency(color))
	err := ColorMap.DoFind(int32(color), func(color, gIdx int32) error {
		b := BuildEmbedding(len(sg.V), len(sg.E)).Fillable()
		b.SetVertex(0, int(color), int(gIdx))
		embs = append(embs, b)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return embs, nil
}

// this is really a depth first search from the given idx
func (sg *SubGraph) EdgeChain() []*Edge {
	edges := make([]*Edge, 0, len(sg.E))
	added := make(map[int]bool, len(sg.E))
	seen := make(map[int]bool, len(sg.V))
	var visit func(int)
	var tryVisit func(int)
	tryVisit = func(u int) {
		if !seen[u] {
			visit(u)
		}
	}
	visit = func(u int) {
		seen[u] = true
		for _, e := range sg.Adj[u] {
			if !added[e] {
				added[e] = true
				edges = append(edges, &sg.E[e])
			}
		}
		toTry := set.NewSortedSet(len(sg.Adj[u])*2)
		for _, e := range sg.Adj[u] {
			toTry.Add(types.Int(sg.E[e].Src))
			toTry.Add(types.Int(sg.E[e].Targ))
		}
		// errors.Logf("EDGE-CHAIN-DEBUG", "%v toTry %v", u, toTry)
		for x, next := toTry.Items()(); next != nil; x, next = next() {
			tryVisit(int(x.(types.Int)))
		}
	}
	visit(0)
	// errors.Logf("DEBUG", "edge chain seen %v", seen)
	// errors.Logf("DEBUG", "edge chain added %v", added)
	// errors.Logf("DEBUG", "edge chain added %v", added)
	return edges
}

func (sg *SubGraph) ExtendEmbedding(G *goiso.Graph, cur *FillableEmbeddingBuilder, e *Edge) []*FillableEmbeddingBuilder {
	// errors.Logf("DEBUG", "extend emb %v with %v", cur.Label(), e)
	// exts := ext.NewCollector(-1)
	exts := make([]*FillableEmbeddingBuilder, 0, 10)

	src := cur.V[e.Src].Idx
	targ := cur.V[e.Targ].Idx

	if src == -1 && targ == -1 {
		panic("src and targ == -1. Which means the edge chain was not connected.")
	} else if src != -1 && targ != -1 {
		// both src and targ are in the builder so we can just add this edge
		exts = append(exts, cur.Copy().Ctx(func(b *FillableEmbeddingBuilder) {
			b.AddEdge(&cur.V[e.Src], &cur.V[e.Targ], e.Color)
		}))
	} else if src != -1 {
		for _, ke := range sg.findEdgesFromSrc(G, cur, src, e) {
			exts = append(exts, cur.Copy().Ctx(func(b *FillableEmbeddingBuilder) {
				b.SetVertex(e.Targ, G.V[ke.Targ].Color, ke.Targ)
				b.AddEdge(&b.V[e.Src], &b.V[e.Targ], e.Color)
			}))
		}
	} else if targ != -1 {
		for _, pe := range sg.findEdgesToTarg(G, cur, targ, e) {
			exts = append(exts, cur.Copy().Ctx(func(b *FillableEmbeddingBuilder) {
				b.SetVertex(e.Src, G.V[pe.Src].Color, pe.Src)
				b.AddEdge(&b.V[e.Src], &b.V[e.Targ], e.Color)
			}))
		}
	} else {
		panic("unreachable")
	}
	return exts
}

func (sg *SubGraph) findEdgesFromSrc(G *goiso.Graph, cur *FillableEmbeddingBuilder, srcId int, e *Edge) []*goiso.Edge {
	tcolor := sg.V[e.Targ].Color
	ecolor := e.Color
	edges := make([]*goiso.Edge, 0, 10)
	for _, ke := range G.Kids[srcId] {
		if ke.Color != ecolor {
			continue
		} else if G.V[ke.Targ].Color != tcolor {
			continue
		} else if cur.HasId(ke.Targ) {
			continue
		}
		edges = append(edges, ke)
	}
	return edges
}

func (sg *SubGraph) findEdgesToTarg(G *goiso.Graph, cur *FillableEmbeddingBuilder, targId int, e *Edge) []*goiso.Edge {
	scolor := sg.V[e.Src].Color
	ecolor := e.Color
	edges := make([]*goiso.Edge, 0, 10)
	for _, pe := range G.Parents[targId] {
		if pe.Color != ecolor {
			continue
		} else if G.V[pe.Src].Color != scolor {
			continue
		} else if cur.HasId(pe.Src) {
			continue
		}
		edges = append(edges, pe)
	}
	return edges
}





/*

func (sg *SubGraph) ExtendEmbedding(G *goiso.Graph, cur *FillableEmbeddingBuilder, e *Edge) []*FillableEmbeddingBuilder {
	// errors.Logf("DEBUG", "extend emb %v with %v", cur.Label(), e)
	// exts := ext.NewCollector(-1)
	exts := make([]*Embedding, 0, 10)

	srcs := sg.findSrcs(cur, e)
	// errors.Logf("DEBUG", "  srcs %v", srcs)
	seen := make(map[int]bool)
	added := 0
	for _, src := range srcs {
		for _, ke := range sg.findEdgesFromSrc(G, cur, src, e) {
			// errors.Logf("DEBUG", "    ke %v %v", ke.Idx, ke)
			if !seen[ke.Idx] {
				seen[ke.Idx] = true
				// extender.Extend(cur, ke, exts.Ch())
				x, _ := cur.EdgeExtend(ke)
				exts = append(exts, x)
				added += 1
			}
		}
	}
	targs := sg.findTargs(cur, e)
	// errors.Logf("DEBUG", "  targs %v", targs)
	for _, targ := range targs {
		for _, pe := range sg.findEdgesFromTarg(G, cur, targ, e) {
			// errors.Logf("DEBUG", "    pe %v %v", pe.Idx, pe)
			if !seen[pe.Idx] {
				seen[pe.Idx] = true
				// extender.Extend(cur, pe, exts.Ch())
				x, _ := cur.EdgeExtend(pe)
				exts = append(exts, x)
				added += 1
			}
		}
	}
	// return exts.Wait(added)
	return exts
}


func (sg *SubGraph) findSrcs(cur *Embedding, e *Edge) []int {
	color := sg.V[e.Src].Color
	srcs := make([]int, 0, 10)
	for i := range cur.V {
		if cur.V[i].Color == color {
			srcs = append(srcs, i)
		}
	}
	return srcs
}

func (sg *SubGraph) findTargs(cur *Embedding, e *Edge) []int {
	color := sg.V[e.Targ].Color
	targs := make([]int, 0, 10)
	for i := range cur.V {
		if cur.V[i].Color == color {
			targs = append(targs, i)
		}
	}
	return targs
}

func (sg *SubGraph) findEdgesFromSrc(G *goiso.Graph, cur *Embedding, src int, e *Edge) []*goiso.Edge {
	srcDtIdx := cur.V[src].Id
	tcolor := sg.V[e.Targ].Color
	ecolor := e.Color
	edges := make([]*goiso.Edge, 0, 10)
	for _, ke := range G.Kids[srcDtIdx] {
		if ke.Color != ecolor {
			continue
		} else if G.V[ke.Targ].Color != tcolor {
			continue
		}
		if !cur.HasEdge(goiso.ColoredArc{ke.Arc, ke.Color}) {
			edges = append(edges, ke)
		}
	}
	return edges
}

func (sg *SubGraph) findEdgesFromTarg(G *goiso.Graph, cur *Embedding, targ int, e *Edge) []*goiso.Edge {
	targDtIdx := cur.V[targ].Id
	scolor := sg.V[e.Src].Color
	ecolor := e.Color
	edges := make([]*goiso.Edge, 0, 10)
	for _, pe := range G.Parents[targDtIdx] {
		if pe.Color != ecolor {
			continue
		} else if G.V[pe.Src].Color != scolor {
			continue
		}
		if !cur.HasEdge(goiso.ColoredArc{pe.Arc, pe.Color}) {
			edges = append(edges, pe)
		}
	}
	return edges
}

*/
