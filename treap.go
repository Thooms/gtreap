package gtreap

import (
	"sync"
)

var nodePool *sync.Pool

func init() {
	nodePool = &sync.Pool{
			New: func() interface{} { return &node{} },
	}
	for i := 0; i < 10000; i++ {
		nodePool.Put(&node{})
	}
}

type Treap struct {
	compare Compare
	root    *node
}

// Compare returns an integer comparing the two items
// lexicographically. The result will be 0 if a==b, -1 if a < b, and
// +1 if a > b.
type Compare func(a, b interface{}) int

// Item can be anything.
type Item interface{}

type node struct {
	item     Item
	priority int
	left     *node
	right    *node
}

func (n *node) reuseWith(item Item, priority int, left, right *node) *node {
	n.item = item
	n.priority = priority
	n.left = left
	n.right = right
	return n
}

func NewTreap(c Compare) *Treap {
	return &Treap{
		compare: c,
		root: nil,
	}
}

func newNode(item Item, priority int, left, right *node) *node {
	n := nodePool.Get().(*node)
	if n == nil {
		n = &node{}
	}
	n.item = item
	n.priority = priority
	n.left = left
	n.right = right

	return n
}

func (t *Treap) Min() Item {
	n := t.root
	if n == nil {
		return nil
	}
	for n.left != nil {
		n = n.left
	}
	return n.item
}

func (t *Treap) Max() Item {
	n := t.root
	if n == nil {
		return nil
	}
	for n.right != nil {
		n = n.right
	}
	return n.item
}

func (t *Treap) Get(target Item) Item {
	n := t.root
	for n != nil {
		c := t.compare(target, n.item)
		if c < 0 {
			n = n.left
		} else if c > 0 {
			n = n.right
		} else {
			return n.item
		}
	}
	return nil
}

// Note: only the priority of the first insert of an item is used.
// Priorities from future updates on already existing items are
// ignored.  To change the priority for an item, you need to do a
// Delete then an Upsert.
func (t *Treap) Upsert(item Item, itemPriority int) *Treap {
	r := t.union(t.root, newNode(item, itemPriority, nil, nil))
	return &Treap{compare: t.compare, root: r}
}

func (t *Treap) union(this *node, that *node) *node {
	if this == nil {
		return that
	}
	if that == nil {
		return this
	}
	if this.priority > that.priority {
		i, p, l, r := this.item, this.priority, this.left, this.right

		left, middle, right := t.split(that, i)

		if middle == nil {
			//return this.reuseWith(i, p, t.union(l, left), t.union(r, right))

			return newNode(i, p, t.union(l, left), t.union(r, right))
			// return &node{
			//	item:     i,
			//	priority: p,
			//	left:     t.union(l, left),
			//	right:    t.union(r, right),
			// }
		}
		return newNode(middle.item, p, t.union(l, left), t.union(r, right))
		// return &node{
		//	item:     middle.item,
		//	priority: p,
		//	left:     t.union(l, left),
		//	right:    t.union(r, right),
		// }
	}

	i, p, l, r := that.item, that.priority, that.left, that.right

	// We don't use middle because the "that" has precendence.
	left, middle, right := t.split(this, i)
	if middle != nil {
		nodePool.Put(middle)
	}

	return newNode(i, p, t.union(left, l), t.union(right, r))

	// &node{
	//	item:     i,
	//	priority: p,
	//	left:     t.union(left, l),
	//	right:    t.union(right, r),
	// }
}

// Splits a treap into two treaps based on a split item "s".
// The result tuple-3 means (left, X, right), where X is either...
// nil - meaning the item s was not in the original treap.
// non-nil - returning the node that had item s.
// The tuple-3's left result treap has items < s,
// and the tuple-3's right result treap has items > s.
func (t *Treap) split(n *node, s Item) (*node, *node, *node) {
	if n == nil {
		return nil, nil, nil
	}
	c := t.compare(s, n.item)
	if c == 0 {
		return n.left, n, n.right
	}
	if c < 0 {
		left, middle, right := t.split(n.left, s)

		return left, middle, newNode(n.item, n.priority, right, n.right)
		// &node{
		//	item:     n.item,
		//	priority: n.priority,
		//	left:     right,
		//	right:    n.right,
		// }
	}
	left, middle, right := t.split(n.right, s)
	return newNode(n.item, n.priority, n.left, left), middle, right

	// &node{
	//	item:     n.item,
	//	priority: n.priority,
	//	left:     n.left,
	//	right:    left,
	// }, middle, right
}

func (t *Treap) Delete(target Item) *Treap {
	left, middle, right := t.split(t.root, target)
	defer nodePool.Put(middle)
	defer nodePool.Put(t.root)
	return &Treap{compare: t.compare, root: t.join(left, right)}
}

// All the items from this are < items from that.
func (t *Treap) join(this *node, that *node) *node {
	if this == nil {
		return that
	}
	if that == nil {
		return this
	}
	if this.priority > that.priority {
		return &node{
			item:     this.item,
			priority: this.priority,
			left:     this.left,
			right:    t.join(this.right, that),
		}
	}
	return &node{
		item:     that.item,
		priority: that.priority,
		left:     t.join(this, that.left),
		right:    that.right,
	}
}

type ItemVisitor func(i Item) bool

// Visit items greater-than-or-equal to the pivot.
func (t *Treap) VisitAscend(pivot Item, visitor ItemVisitor) {
	t.visitAscend(t.root, pivot, visitor)
}

func (t *Treap) visitAscend(n *node, pivot Item, visitor ItemVisitor) bool {
	if n == nil {
		return true
	}
	if t.compare(pivot, n.item) <= 0 {
		if !t.visitAscend(n.left, pivot, visitor) {
			return false
		}
		if !visitor(n.item) {
			return false
		}
	}
	return t.visitAscend(n.right, pivot, visitor)
}
