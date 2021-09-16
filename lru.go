package high_performance_lru

import (
	"context"
	"errors"
)

const (
	MaxElem     = 1000
	MinElem     = 10
	DefaultElem = 100
)

var (
	EmptyErr    = errors.New("[fastlru] lur is empty")
	NotFoundErr = errors.New("[fastlru] not found")
	UnknowErr   = errors.New("[fastlru] unknow error")
)

type Lru interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, val interface{}) error
	Clear(ctx context.Context)
	GetAllValue(ctx context.Context) []interface{}
}

type Node struct {
	elem *Elem
	next *Node
}

type Elem struct {
	key   string
	value interface{}
	node  *Node
}

type lruCache struct {
	maxCount, current int
	keyMap            map[string]*Elem
	first, head, tail *Node
}

func NewLru(maxCount int) Lru {
	count := maxCount
	if count < MinElem {
		count = MinElem
	} else if count > MaxElem {
		count = MaxElem
	}

	emptyElem := &Node{}
	return &lruCache{
		keyMap:   make(map[string]*Elem, count),
		first:    emptyElem,
		maxCount: maxCount,
	}

	return nil
}

func (l *lruCache) Get(ctx context.Context, key string) (interface{}, error) {
	// 1.如果lru为空则查找失败
	if l.empty() {
		return nil, EmptyErr
	}

	// 2.不为空
	elem, ok := l.keyMap[key]
	if !ok {
		return nil, NotFoundErr
	}

	// 3.找到了，返回对应的值，并且把当前元素移动到队首
	val := elem.value
	l.moveElem2Header(ctx, elem)
	return val, nil
}

func (l *lruCache) Set(ctx context.Context, key string, val interface{}) error {
	// 1.查找
	v, err := l.Get(ctx, key)
	if err == nil {
		// 找到了，则替换第一个元素
		if val != v {
			l.head.elem.value = val
		}
	} else if err == EmptyErr {
		l.insertFirstElem(ctx, key, val)
	} else if err == NotFoundErr {
		l.insertElem(ctx, key, val)
	}

	return nil
}

func (l *lruCache) Clear(ctx context.Context) {
	l.keyMap = make(map[string]*Elem, l.maxCount)
	l.first = nil
	l.head = nil
	l.tail = nil
	l.current = 0
	return
}

func (l *lruCache) GetAllValue(ctx context.Context) []interface{} {
	result := make([]interface{}, 0, len(l.keyMap))
	p := l.head
	for p != nil {
		result = append(result, p.elem.value)
		p = p.next
	}
	return result
}

func (l *lruCache) empty() bool {
	return l.current == 0
}

// 将指定元素移动到队首
func (l *lruCache) moveElem2Header(ctx context.Context, elem *Elem) {
	if elem == l.head.elem {
		return
	}

	newFirstNode := elem.node.next
	if newFirstNode.next == nil {
		// 最后一个元素
		l.tail = elem.node
		l.tail.next = nil // mark:这里漏了，找了一个多小时
	} else {
		newFirstNode.elem.node.next = newFirstNode.next
		newFirstNode.next.elem.node = newFirstNode.elem.node
		newFirstNode.elem.node = nil
	}
	// 最后一个元素插入
	newFirstNode.next = l.head
	l.head.elem.node = newFirstNode // mark:这一行漏了，调试2个多小时才发现
	l.head = newFirstNode
	l.first.next = newFirstNode
	newFirstNode.elem = elem
	elem.node = l.first
}

// 插入第一个元素
func (l *lruCache) insertFirstElem(ctx context.Context, key string, val interface{}) {
	elem := &Elem{
		key:   key,
		value: val,
		node:  l.first,
	}
	node := &Node{
		elem: elem,
	}
	l.keyMap[key] = elem
	l.first.next = node
	l.head = node
	l.tail = node
	l.current++
}

// 插入元素
func (l *lruCache) insertElem(ctx context.Context, key string, val interface{}) {
	if l.current == l.maxCount {
		// lru已满，需要淘汰数据
		l.deleteTail(ctx)
	}
	// 向队首插入一个元素
	elem := &Elem{
		key:   key,
		value: val,
		node:  l.first,
	}
	l.keyMap[key] = elem
	node := &Node{
		elem: elem,
		next: l.head,
	}
	// todo
	l.head.elem.node = node
	l.first.next = node
	l.head = node
	l.current++
}

func (l *lruCache) deleteTail(ctx context.Context) {
	tmpNode := l.tail.elem.key
	l.tail.elem.node.next = nil
	l.tail = l.tail.elem.node
	delete(l.keyMap, tmpNode)
	l.current--
}
