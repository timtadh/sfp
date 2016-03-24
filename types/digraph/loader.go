package digraph

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"runtime"
	"strings"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/data-structures/hashtable"
	"github.com/timtadh/data-structures/types"
	"github.com/timtadh/goiso"
)

import (
	"github.com/timtadh/sfp/config"
	"github.com/timtadh/sfp/lattice"
	"github.com/timtadh/sfp/stores/bytes_bytes"
	"github.com/timtadh/sfp/stores/bytes_int"
	"github.com/timtadh/sfp/stores/bytes_subgraph"
	"github.com/timtadh/sfp/stores/int_edge"
	"github.com/timtadh/sfp/stores/int_int"
	"github.com/timtadh/sfp/stores/int_json"
	"github.com/timtadh/sfp/types/digraph/ext"
	"github.com/timtadh/sfp/types/digraph/subgraph"
)

type ErrorList []error

func (self ErrorList) Error() string {
	var s []string
	for _, err := range self {
		s = append(s, err.Error())
	}
	return "Errors [" + strings.Join(s, ", ") + "]"
}

type Digraph struct {
	MinEdges, MaxEdges       int
	MinVertices, MaxVertices int
	G                        *goiso.Graph
	FrequentVertices         [][]byte
	Supported                Supported
	Extender                 *ext.Extender
	NodeAttrs                int_json.MultiMap
	Embeddings               bytes_subgraph.MultiMap
	Parents                  bytes_bytes.MultiMap
	ParentCount              bytes_int.MultiMap
	Children                 bytes_bytes.MultiMap
	ChildCount               bytes_int.MultiMap
	CanonKids                bytes_bytes.MultiMap
	CanonKidCount            bytes_int.MultiMap
	ColorMap                 int_int.MultiMap
	ColorInEdges            int_edge.MultiMap
	ColorOutEdges           int_edge.MultiMap
	config                   *config.Config
	search                   bool
}

func NewDigraph(config *config.Config, search bool, sup Supported, minE, maxE, minV, maxV int) (g *Digraph, err error) {
	nodeAttrs, err := config.IntJsonMultiMap("digraph-node-attrs")
	if err != nil {
		return nil, err
	}
	parents, err := config.MultiMap("digraph-parents")
	if err != nil {
		return nil, err
	}
	parentCount, err := config.BytesIntMultiMap("digraph-parent-count")
	if err != nil {
		return nil, err
	}
	children, err := config.MultiMap("digraph-children")
	if err != nil {
		return nil, err
	}
	childCount, err := config.BytesIntMultiMap("digraph-child-count")
	if err != nil {
		return nil, err
	}
	canonKids, err := config.MultiMap("digraph-canon-kids")
	if err != nil {
		return nil, err
	}
	canonKidCount, err := config.BytesIntMultiMap("digraph-canon-kid-count")
	if err != nil {
		return nil, err
	}
	colorMap, err := config.IntIntMultiMap("digraph-color-map")
	if err != nil {
		return nil, err
	}
	colorInEdges, err := config.IntEdgeMultiMap("digraph-color-in-edges")
	if err != nil {
		return nil, err
	}
	colorOutEdges, err := config.IntEdgeMultiMap("digraph-color-out-edges")
	if err != nil {
		return nil, err
	}
	g = &Digraph{
		Supported:     sup,
		Extender:      ext.NewExtender(runtime.NumCPU()),
		MinEdges:      minE,
		MaxEdges:      maxE,
		MinVertices:   minV,
		MaxVertices:   maxV,
		NodeAttrs:     nodeAttrs,
		Parents:       parents,
		ParentCount:   parentCount,
		Children:      children,
		ChildCount:    childCount,
		CanonKids:     canonKids,
		CanonKidCount: canonKidCount,
		ColorMap:      colorMap,
		ColorInEdges:  colorInEdges,
		ColorOutEdges: colorOutEdges,
		config:        config,
		search:        search,
	}
	return g, nil
}

func (g *Digraph) Support() int {
	return g.config.Support
}

func (g *Digraph) LargestLevel() int {
	return g.MaxEdges
}

func (g *Digraph) MinimumLevel() int {
	if g.MinEdges > 0 {
		return g.MinEdges
	} else if g.MinVertices > 0 {
		return g.MinVertices - 1
	}
	return 0
}

func RootSearchNode(g *Digraph) *SearchNode {
	return NewSearchNode(g, nil)
}

func RootEmbListNode(g *Digraph) *EmbListNode {
	return NewEmbListNode(g, nil)
}

func (g *Digraph) Root() lattice.Node {
	if g.search {
		return RootSearchNode(g)
	} else {
		return RootEmbListNode(g)
	}
}

func VE(node lattice.Node) (V, E int) {
	E = 0
	V = 0
	switch n := node.(type) {
	case *EmbListNode:
		if len(n.sgs) > 0 {
			E = len(n.sgs[0].E)
			V = len(n.sgs[0].V)
		}
	case *SearchNode:
		E = len(n.Pat.E)
		V = len(n.Pat.V)
	default:
		panic(errors.Errorf("unknown node type %T %v", node, node))
	}
	return V, E
}

func (g *Digraph) Acceptable(node lattice.Node) bool {
	V, E := VE(node)
	return g.MinEdges <= E && E <= g.MaxEdges && g.MinVertices <= V && V <= g.MaxVertices
}

func (g *Digraph) TooLarge(node lattice.Node) bool {
	V, E := VE(node)
	return E > g.MaxEdges || V > g.MaxVertices
}

func (g *Digraph) Close() error {
	g.config.AsyncTasks.Wait()
	g.Extender.Stop()
	g.NodeAttrs.Close()
	g.Embeddings.Close()
	g.Parents.Close()
	g.ParentCount.Close()
	g.Children.Close()
	g.ChildCount.Close()
	g.CanonKids.Close()
	g.CanonKidCount.Close()
	g.ColorMap.Close()
	g.ColorInEdges.Close()
	g.ColorOutEdges.Close()
	return nil
}

type VegLoader struct {
	dt *Digraph
}

func NewVegLoader(config *config.Config, search bool, sup Supported, minE, maxE, minV, maxV int) (lattice.Loader, error) {
	g, err := NewDigraph(config, search, sup, minE, maxE, minV, maxV)
	if err != nil {
		return nil, err
	}
	v := &VegLoader{
		dt: g,
	}
	return v, nil
}

func (v *VegLoader) Load(input lattice.Input) (dt lattice.DataType, err error) {
	G, err := v.loadDigraph(input)
	if err != nil {
		return nil, err
	}
	err = v.dt.Init(G)
	if err != nil {
		return nil, err
	}
	return v.dt, nil
}

func (dt *Digraph) Init(G *goiso.Graph) (err error) {
	dt.G = G
	dt.Embeddings, err = dt.config.BytesSubgraphMultiMap("digraph-embeddings", bytes_subgraph.DeserializeSubGraph(G))
	if err != nil {
		return err
	}

	for i := range G.V {
		u := &G.V[i]
		err = dt.ColorMap.Add(int32(u.Color), int32(u.Idx))
		if err != nil {
			return err
		}
		if G.ColorFrequency(u.Color) >= dt.config.Support {
			sg, _ := G.VertexSubGraph(u.Idx)
			err := dt.Embeddings.Add(sg.ShortLabel(), sg)
			if err != nil {
				return err
			}
		}
	}

	for i := range G.E {
		x := &G.E[i]
		e := subgraph.Edge{
			Src:G.V[x.Src].Color,
			Targ:G.V[x.Targ].Color,
			Color:x.Color,
		}
		if G.ColorFrequency(e.Src) < dt.config.Support {
			continue
		}
		if G.ColorFrequency(e.Targ) < dt.config.Support {
			continue
		}
		if G.ColorFrequency(e.Color) < dt.config.Support {
			continue
		}
		err = dt.ColorOutEdges.Add(int32(e.Src), e)
		if err != nil {
			return err
		}
		err = dt.ColorInEdges.Add(int32(e.Targ), e)
		if err != nil {
			return err
		}
	}

	err = bytes_subgraph.DoKey(dt.Embeddings.Keys, func(label []byte) error {
		dt.FrequentVertices = append(dt.FrequentVertices, label)
		return nil
	})

	return nil
}

func (v *VegLoader) loadDigraph(input lattice.Input) (graph *goiso.Graph, err error) {
	var errs ErrorList
	V, E, err := graphSize(input)
	if err != nil {
		return nil, err
	}
	G := goiso.NewGraph(V, E)
	graph = &G
	vids := hashtable.NewLinearHash() // int64 ==> *goiso.Vertex

	in, closer := input()
	defer closer()
	err = processLines(in, func(line []byte) {
		if len(line) == 0 || !bytes.Contains(line, []byte("\t")) {
			return
		}
		line_type, data := parseLine(line)
		switch line_type {
		case "vertex":
			if err := v.loadVertex(graph, vids, data); err != nil {
				errs = append(errs, err)
			}
		case "edge":
			if err := v.loadEdge(graph, vids, data); err != nil {
				errs = append(errs, err)
			}
		default:
			errs = append(errs, errors.Errorf("Unknown line type %v", line_type))
		}
	})
	if err != nil {
		return nil, err
	}
	if len(errs) == 0 {
		return graph, nil
	}
	return graph, errs
}

func (v *VegLoader) loadVertex(g *goiso.Graph, vids types.Map, data []byte) (err error) {
	obj, err := parseJson(data)
	if err != nil {
		return err
	}
	_id, err := obj["id"].(json.Number).Int64()
	if err != nil {
		return err
	}
	label := strings.TrimSpace(obj["label"].(string))
	id := int(_id)
	vertex := g.AddVertex(id, label)
	err = vids.Put(types.Int(id), vertex)
	if err != nil {
		return err
	}
	err = v.dt.NodeAttrs.Add(int32(vertex.Id), obj)
	if err != nil {
		return err
	}
	return nil
}

func (v *VegLoader) loadEdge(g *goiso.Graph, vids types.Map, data []byte) (err error) {
	obj, err := parseJson(data)
	if err != nil {
		return err
	}
	_src, err := obj["src"].(json.Number).Int64()
	if err != nil {
		return err
	}
	_targ, err := obj["targ"].(json.Number).Int64()
	if err != nil {
		return err
	}
	src := int(_src)
	targ := int(_targ)
	label := strings.TrimSpace(obj["label"].(string))
	if o, err := vids.Get(types.Int(src)); err != nil {
		return err
	} else {
		u := o.(*goiso.Vertex)
		if o, err := vids.Get(types.Int(targ)); err != nil {
			return err
		} else {
			v := o.(*goiso.Vertex)
			g.AddEdge(u, v, label)
		}
	}
	return nil
}

func processLines(in io.Reader, process func([]byte)) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		unsafe := scanner.Bytes()
		line := make([]byte, len(unsafe))
		copy(line, unsafe)
		process(line)
	}
	return scanner.Err()
}

func parseJson(data []byte) (obj map[string]interface{}, err error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func parseLine(line []byte) (line_type string, data []byte) {
	split := bytes.Split(line, []byte("\t"))
	return strings.TrimSpace(string(split[0])), bytes.TrimSpace(split[1])
}

func graphSize(input lattice.Input) (V, E int, err error) {
	in, closer := input()
	defer closer()
	err = processLines(in, func(line []byte) {
		if bytes.HasPrefix(line, []byte("vertex")) {
			V++
		} else if bytes.HasPrefix(line, []byte("edge")) {
			E++
		}
	})
	if err != nil {
		return 0, 0, err
	}
	return V, E, nil
}
