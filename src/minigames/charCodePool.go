package minigames

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
)

type SymbolSet struct {
	Lowercase []rune
	Uppercase []rune
}
type symbols struct {
	English SymbolSet
	Danish  SymbolSet
}

var SymbolSets = symbols{
	English: SymbolSet{
		Lowercase: []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z'},
		Uppercase: []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z'},
	},
	Danish: SymbolSet{
		Lowercase: []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', 'æ', 'ø', 'å'},
		Uppercase: []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z', 'Æ', 'Ø', 'Å'},
	},
}

func NewCharCodePool(initialSize uint32, charCodeLength uint32, symbols SymbolSet) (*CharCodePool, error) {
	var numUniqueSymbols = len(symbols.Lowercase) + len(symbols.Uppercase)
	var possiblePermutations = math.Pow(float64(numUniqueSymbols), float64(charCodeLength))
	if possiblePermutations < float64(initialSize) {
		return nil, fmt.Errorf("initialSize %d is larger than the number of possible permutations %f", initialSize, possiblePermutations)
	}

	charPool := NewCharPool(symbols)
	codePool := &CharCodePool{
		codeLength: charCodeLength,
		charPool:   charPool,
		codePool:   make([]PoolEntry[[]rune], initialSize),
	}
	for i := uint32(0); i < initialSize; i++ {
		code := make([]rune, charCodeLength)
		for j := 0; j < len(code); j++ {
			code[j] = charPool.GetNextChar()
		}
		codePool.codePool[i] = *NewPoolEntry(code, codePool)
	}

	return codePool, nil
}

func NewPoolEntry[T any](value T, pool Pool[T]) *PoolEntry[T] {
	return &PoolEntry[T]{
		Value: value,
		pool:  pool,
	}
}

type PoolEntry[T any] struct {
	Value T
	pool  Pool[T]
}

// Returns code to pool
func (pe *PoolEntry[T]) Free() {
	pe.pool.Reintroduce(pe)
}

type Pool[T any] interface {
	GetNext() *PoolEntry[T]
	Reintroduce(*PoolEntry[T])
}

type CharCodePool struct {
	sync.Mutex
	codeLength uint32
	codePool   []PoolEntry[[]rune]
	charPool   *CharPool
}

// Xi Shing Ping intensifies
func (ccp *CharCodePool) GetNext() *PoolEntry[[]rune] {
	ccp.Lock()
	defer ccp.Unlock()
	if len(ccp.codePool) == 0 {
		code := make([]rune, ccp.codeLength)
		for i := uint32(0); i < ccp.codeLength; i++ {
			code[i] = ccp.charPool.GetNextChar()
		}
		return NewPoolEntry(code, ccp)
	}
	entry := ccp.codePool[len(ccp.codePool)-1]
	ccp.codePool = ccp.codePool[:len(ccp.codePool)-1]
	return &entry
}

func (ccp *CharCodePool) Reintroduce(pe *PoolEntry[[]rune]) {
	ccp.Lock()
	defer ccp.Unlock()
	ccp.codePool = append(ccp.codePool, *pe)
}

func NewCharPool(symbolsSet SymbolSet) *CharPool {
	//Allocate shared symbols array
	var symbols = make([]rune, len(symbolsSet.Lowercase)+len(symbolsSet.Uppercase))

	//Copy symbols into shared array
	copy(symbols, symbolsSet.Lowercase)
	copy(symbols[len(symbolsSet.Lowercase):], symbolsSet.Uppercase)

	//Shuffle symbols
	rand.Shuffle(len(symbols), func(i, j int) {
		symbols[i], symbols[j] = symbols[j], symbols[i]
	})

	return &CharPool{
		indexPointer: 0,
		symbols:      symbols,
	}
}

// Threadsafe
//
// Actually not a pool as known from the Pool interface, but a pool of characters
// that assures that any character drawn, is random, and that all characters are
// drawn before any one character is drawn again.
type CharPool struct {
	sync.Mutex
	indexPointer uint32
	symbols      []rune
}

func (cp *CharPool) GetNextChar() rune {
	cp.Lock()
	defer cp.Unlock()
	if cp.indexPointer >= uint32(len(cp.symbols)) {
		cp.indexPointer = 0
		rand.Shuffle(len(cp.symbols), func(i, j int) {
			cp.symbols[i], cp.symbols[j] = cp.symbols[j], cp.symbols[i]
		})
	}
	nextChar := cp.symbols[cp.indexPointer]
	cp.indexPointer++
	return nextChar
}
