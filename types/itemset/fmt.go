package itemset

import (
	"fmt"
	"io"
	"strings"
)

import (
	"github.com/timtadh/sfp/lattice"
)

type Formatter struct{}

func (f Formatter) FileExt() string {
	return ".items"
}

func (f Formatter) Pattern(node lattice.Node) (string, error) {
	n := node.(*Node)
	items := make([]string, 0, n.pat.Items.Size())
	for i, next := n.pat.Items.Items()(); next != nil; i, next = next() {
		items = append(items, fmt.Sprintf("%v", i))
	}
	return fmt.Sprintf("%s", strings.Join(items, " ")), nil
}

func (f Formatter) Embeddings(node lattice.Node) (string, error) {
	n := node.(*Node)
	txs := make([]string, 0, len(n.txs))
	for _, tx := range n.txs {
		txs = append(txs, fmt.Sprintf("%v", tx))
	}
	pat, err := f.Pattern(node)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s : %s", pat, strings.Join(txs, " ")), nil
}

func (f Formatter) FormatPattern(w io.Writer, node lattice.Node) error {
	n := node.(*Node)
	pat, err := f.Pattern(node)
	if err != nil {
		return err
	}
	max := ""
	if ismax, err := n.Maximal(); err != nil {
		return err
	} else if ismax {
		max = " # maximal"
	}
	_, err = fmt.Fprintf(w, "%s%s\n", pat, max)
	return err
}

func (f Formatter) FormatEmbeddings(w io.Writer, node lattice.Node) error {
	emb, err := f.Embeddings(node)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", emb)
	return err
}
