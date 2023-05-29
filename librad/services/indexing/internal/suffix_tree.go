package internal

import "sort"

// 后缀索引，string->T的映射，支持模糊键值搜索
type SuffixTree struct {
	Num  int32
	Ends []string
	Sons map[rune]*SuffixTree
}

func NewSuffixTree() *SuffixTree {
	return &SuffixTree{
		Sons: make(map[rune]*SuffixTree),
	}
}

func (tree *SuffixTree) add(a []rune, s string) bool {
	if len(a) == 0 {
		i := sort.SearchStrings(tree.Ends, s)
		if i < len(tree.Ends) && tree.Ends[i] == s {
			return false
		}
		tree.Ends = append(tree.Ends, "")
		copy(tree.Ends[i+1:], tree.Ends[i:])
		tree.Ends[i] = s
		tree.Num++
		return true
	}
	son, ok := tree.Sons[a[0]]
	if !ok {
		son = NewSuffixTree()
		tree.Sons[a[0]] = son
	}
	if !son.add(a[1:], s) {
		return false
	}
	tree.Num++
	return true
}

func (tree *SuffixTree) del(a []rune, s string) bool {
	if len(a) == 0 {
		i := sort.SearchStrings(tree.Ends, s)
		if !(i < len(tree.Ends) && tree.Ends[i] == s) {
			return false
		}
		tree.Ends = append(tree.Ends[:i], tree.Ends[i+1:]...)
		tree.Num--
		return true
	}
	son, ok := tree.Sons[a[0]]
	if !ok || !son.del(a[1:], s) {
		return false
	}
	if tree.Num--; son.Num == 0 {
		delete(tree.Sons, a[0])
	}
	return true
}

// 结果需要去重
func (tree *SuffixTree) search(a []rune, m map[string]struct{}) {
	if len(a) == 0 {
		for _, s := range tree.Ends {
			m[s] = struct{}{}
		}
		for _, son := range tree.Sons {
			son.search(a, m)
		}
		return
	}
	son, ok := tree.Sons[a[0]]
	if !ok {
		return
	}
	son.search(a[1:], m)
}

func (tree *SuffixTree) Add(s string) bool {
	a := []rune(s)
	if !tree.add(a, s) {
		return false
	}
	for i := 1; i < len(a); i++ {
		tree.add(a[i:], s)
	}
	return true
}

func (tree *SuffixTree) Del(s string) bool {
	a := []rune(s)
	if tree.del(a, s) {
		return false
	}
	for i := 1; i < len(a); i++ {
		tree.del(a[i:], s)
	}
	return true
}

func (tree *SuffixTree) Search(s string) (r []string) {
	m := make(map[string]struct{})
	tree.search([]rune(s), m)
	for s := range m {
		r = append(r, s)
	}
	return
}
