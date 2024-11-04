package util

import (
	"math"
	"sync"
	"testing"
	"unicode"
)

func TestNewCharPool(t *testing.T) {
	englishPool := NewCharPool(SymbolSets.English)
	if englishPool == nil {
		t.Fatal("NewCharPool returned nil for English SymbolSet")
	}
	if len(englishPool.symbols) != 52 {
		t.Errorf("Expected 52 symbols in English pool, got %d", len(englishPool.symbols))
	}

	danishPool := NewCharPool(SymbolSets.Danish)
	if danishPool == nil {
		t.Fatal("NewCharPool returned nil for Danish SymbolSet")
	}
	if len(danishPool.symbols) != 58 {
		t.Errorf("Expected 58 symbols in Danish pool, got %d", len(danishPool.symbols))
	}
}

func TestGetNextChar(t *testing.T) {
	pool := NewCharPool(SymbolSets.English)
	totalSymbols := len(SymbolSets.English.Lowercase) + len(SymbolSets.English.Uppercase)
	usedChars := make(map[rune]bool)

	// Test that we get all characters before repeating
	for i := 0; i < totalSymbols; i++ {
		char := pool.GetNextChar()
		if usedChars[char] {
			t.Errorf("Character %c repeated before all characters were used", char)
		}
		usedChars[char] = true
	}

	// Test that after 52 characters, we start over
	newChar := pool.GetNextChar()
	if !usedChars[newChar] {
		t.Errorf("Expected to start over after %d characters, but got new character %c", totalSymbols, newChar)
	}
}

func TestCharPoolThreadSafety(t *testing.T) {
	pool := NewCharPool(SymbolSets.English)
	const numGoroutines = 100
	const charsPerGoroutine = 1000

	results := make(chan []rune, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			chars := make([]rune, charsPerGoroutine)
			for j := 0; j < charsPerGoroutine; j++ {
				chars[j] = pool.GetNextChar()
			}
			results <- chars
		}()
	}

	allChars := make(map[rune]int)
	for i := 0; i < numGoroutines; i++ {
		chars := <-results
		for _, char := range chars {
			allChars[char]++
		}
	}

	// Check that we have all characters and they're evenly distributed
	if len(allChars) != 52 {
		t.Errorf("Expected 52 unique characters, got %d", len(allChars))
	}

	expectedCount := numGoroutines * charsPerGoroutine / 52
	for char, count := range allChars {
		if count < expectedCount-100 || count > expectedCount+100 {
			t.Errorf("Character %c has uneven distribution: %d (expected around %d)", char, count, expectedCount)
		}
	}
}

func TestSymbolSets(t *testing.T) {
	// Test English SymbolSet
	if len(SymbolSets.English.Lowercase) != 26 {
		t.Errorf("Expected 26 lowercase English symbols, got %d", len(SymbolSets.English.Lowercase))
	}
	if len(SymbolSets.English.Uppercase) != 26 {
		t.Errorf("Expected 26 uppercase English symbols, got %d", len(SymbolSets.English.Uppercase))
	}

	// Test Danish SymbolSet
	if len(SymbolSets.Danish.Lowercase) != 29 {
		t.Errorf("Expected 29 lowercase Danish symbols, got %d", len(SymbolSets.Danish.Lowercase))
	}
	if len(SymbolSets.Danish.Uppercase) != 29 {
		t.Errorf("Expected 29 uppercase Danish symbols, got %d", len(SymbolSets.Danish.Uppercase))
	}

	// Test that all English symbols are valid letters
	for _, char := range SymbolSets.English.Lowercase {
		if !unicode.IsLetter(char) || !unicode.IsLower(char) {
			t.Errorf("Invalid lowercase English symbol: %c", char)
		}
	}
	for _, char := range SymbolSets.English.Uppercase {
		if !unicode.IsLetter(char) || !unicode.IsUpper(char) {
			t.Errorf("Invalid uppercase English symbol: %c", char)
		}
	}

	// Test that all Danish symbols are valid letters
	for _, char := range SymbolSets.Danish.Lowercase {
		if !unicode.IsLetter(char) || !unicode.IsLower(char) {
			t.Errorf("Invalid lowercase Danish symbol: %c", char)
		}
	}
	for _, char := range SymbolSets.Danish.Uppercase {
		if !unicode.IsLetter(char) || !unicode.IsUpper(char) {
			t.Errorf("Invalid uppercase Danish symbol: %c", char)
		}
	}
}

func TestNewCharCodePool(t *testing.T) {
	tests := []struct {
		name           string
		initialSize    uint32
		charCodeLength uint32
		symbols        SymbolSet
		wantErr        bool
	}{
		{
			name:           "Valid English pool",
			initialSize:    100,
			charCodeLength: 4,
			symbols:        SymbolSets.English,
			wantErr:        false,
		},
		{
			name:           "Valid Danish pool",
			initialSize:    100,
			charCodeLength: 4,
			symbols:        SymbolSets.Danish,
			wantErr:        false,
		},
		{
			name:           "Size test 1",
			initialSize:    52,
			charCodeLength: 1,
			symbols:        SymbolSets.Danish,
			wantErr:        false,
		},
		{
			name:           "Size test 2",
			initialSize:    200,
			charCodeLength: 2,
			symbols:        SymbolSets.Danish,
			wantErr:        false,
		},
		{
			name:           "Size test 3",
			initialSize:    100,
			charCodeLength: 3,
			symbols:        SymbolSets.Danish,
			wantErr:        false,
		},
		{
			name:           "Invalid size 1",
			initialSize:    10000000,
			charCodeLength: 2,
			symbols:        SymbolSets.English,
			wantErr:        true,
		},
		{
			name:           "Invalid size 2",
			initialSize:    56,
			charCodeLength: 1,
			symbols:        SymbolSets.English,
			wantErr:        true,
		},
		{
			name:           "Invalid size 3",
			initialSize:    60,
			charCodeLength: 1,
			symbols:        SymbolSets.Danish,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := NewCharCodePool(tt.initialSize, tt.charCodeLength, tt.symbols)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCharCodePool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if pool == nil {
					t.Errorf("NewCharCodePool() returned nil pool")
				}
				if len(pool.codePool) != int(tt.initialSize) {
					t.Errorf("%s: Expected initial pool size %d, got %d", tt.name, tt.initialSize, len(pool.codePool))
				}
				if pool.codeLength != tt.charCodeLength {
					t.Errorf("Expected code length %d, got %d", tt.charCodeLength, pool.codeLength)
				}
			}
		})
	}
}

func TestCharCodePoolGetNext(t *testing.T) {
	initialSize := uint32(100)
	codeLength := uint32(4)
	pool, err := NewCharCodePool(initialSize, codeLength, SymbolSets.English)
	if err != nil {
		t.Fatalf("Failed to create CharCodePool: %v", err)
	}

	// Test getting all initial codes
	codes := make(map[string]bool)
	for i := 0; i < int(initialSize); i++ {
		entry := pool.GetNext()
		code := string(entry.Value)
		if codes[code] {
			t.Errorf("Duplicate code found: %s", code)
		}
		codes[code] = true
		if len(entry.Value) != int(codeLength) {
			t.Errorf("Expected code length %d, got %d", codeLength, len(entry.Value))
		}
	}

	// Test generating a new code after exhausting the pool
	newEntry := pool.GetNext()
	newCode := string(newEntry.Value)
	if codes[newCode] {
		t.Errorf("New generated code %s already exists in the pool", newCode)
	}
	if len(newEntry.Value) != int(codeLength) {
		t.Errorf("Expected new code length %d, got %d", codeLength, len(newEntry.Value))
	}
}

func TestCharCodePoolReintroduce(t *testing.T) {
	initialSize := uint32(100)
	codeLength := uint32(4)
	pool, _ := NewCharCodePool(initialSize, codeLength, SymbolSets.English)

	// Get a code and reintroduce it
	entry := pool.GetNext()
	pool.Reintroduce(entry)

	// The pool should now have 100 entries again
	if len(pool.codePool) != int(initialSize) {
		t.Errorf("Expected pool size %d after reintroducing, got %d", initialSize, len(pool.codePool))
	}

	// The reintroduced code should be at the end of the pool
	lastEntry := pool.codePool[len(pool.codePool)-1]
	if string(lastEntry.Value) != string(entry.Value) {
		t.Errorf("Expected last code in pool to be %s, got %s", string(entry.Value), string(lastEntry.Value))
	}
}

func TestCharCodePoolPermutationExhaustion(t *testing.T) {
	initialSize := uint32(100)
	codeLength := uint32(2)
	pool, _ := NewCharCodePool(initialSize, codeLength, SymbolSets.English)
	uniqueSymbols := len(SymbolSets.English.Lowercase) + len(SymbolSets.English.Uppercase)
	possiblePermutations := math.Pow(float64(uniqueSymbols), float64(codeLength))

	// Exhaust the pool
	codes := make(map[string]bool)
	for i := 0; i < int(possiblePermutations); i++ {
		entry := pool.GetNext()
		code := string(entry.Value)
		if codes[code] {
			t.Errorf("Duplicate code found: %s", code)
		}
		codes[code] = true
	}

	// The pool should now be empty
	if len(pool.codePool) != 0 {
		t.Errorf("Expected pool to be empty after exhausting, got size %d", len(pool.codePool))
	}

	// Getting a new code should return one that has been returned before
	newEntry := pool.GetNext()
	newCode := string(newEntry.Value)
	if !codes[newCode] {
		t.Errorf("Expected new code %s to have been returned before", newCode)
	}
}

func TestCharCodePoolConcurrency(t *testing.T) {
	initialSize := uint32(1000)
	codeLength := uint32(3)
	pool, _ := NewCharCodePool(initialSize, codeLength, SymbolSets.English)
	uniqueSymbols := len(SymbolSets.English.Lowercase) + len(SymbolSets.English.Uppercase)
	possiblePermutations := math.Pow(float64(uniqueSymbols), float64(codeLength))
	flooredSQRT := int(math.Floor(math.Sqrt(possiblePermutations)))

	var wg sync.WaitGroup
	numGoroutines := flooredSQRT
	codesPerGoroutine := flooredSQRT

	mapMutex := sync.Mutex{}
	codes := map[string]bool{}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < codesPerGoroutine; j++ {
				entry := pool.GetNext()
				code := string(entry.Value)
				mapMutex.Lock()
				if _, loaded := codes[code]; loaded {
					t.Errorf("Concurrent access produced duplicate code: %s", code)
				}
				codes[code] = true
				mapMutex.Unlock()
			}
		}()
	}

	wg.Wait()
}

func TestPoolEntryFree(t *testing.T) {
	pool, _ := NewCharCodePool(100, 4, SymbolSets.English)
	entry := pool.GetNext()

	initialPoolSize := len(pool.codePool)
	entry.Free()

	if len(pool.codePool) != initialPoolSize+1 {
		t.Errorf("Expected pool size to increase by 1 after Free(), got %d", len(pool.codePool))
	}

	lastEntry := pool.codePool[len(pool.codePool)-1]
	if string(lastEntry.Value) != string(entry.Value) {
		t.Errorf("Expected freed entry to be reintroduced to the pool")
	}
}
