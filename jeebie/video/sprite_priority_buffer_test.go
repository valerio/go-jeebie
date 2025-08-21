package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpritePriorityBuffer_Clear(t *testing.T) {
	buffer := &SpritePriorityBuffer{}

	// set some values
	buffer.ownerIndex[0] = 5
	buffer.ownerX[0] = 10
	buffer.ownerIndex[50] = 3
	buffer.ownerX[50] = 20

	// clear should reset everything
	buffer.Clear()

	for i := 0; i < FramebufferWidth; i++ {
		assert.Equal(t, -1, buffer.ownerIndex[i], "pixel %d should have no owner", i)
		assert.Equal(t, 0xFF, buffer.ownerX[i], "pixel %d should have max X value", i)
	}
}

func TestSpritePriorityBuffer_TryClaimPixel(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*SpritePriorityBuffer)
		pixelX        int
		spriteIndex   int
		spriteX       int
		expectedClaim bool
		expectedOwner int
	}{
		{
			name: "claim unowned pixel",
			setup: func(b *SpritePriorityBuffer) {
				b.Clear()
			},
			pixelX:        50,
			spriteIndex:   2,
			spriteX:       20,
			expectedClaim: true,
			expectedOwner: 2,
		},
		{
			name: "lower X coordinate wins",
			setup: func(b *SpritePriorityBuffer) {
				b.Clear()
				b.ownerIndex[50] = 3
				b.ownerX[50] = 30
			},
			pixelX:        50,
			spriteIndex:   2,
			spriteX:       20, // lower X wins
			expectedClaim: true,
			expectedOwner: 2,
		},
		{
			name: "higher X coordinate loses",
			setup: func(b *SpritePriorityBuffer) {
				b.Clear()
				b.ownerIndex[50] = 3
				b.ownerX[50] = 10
			},
			pixelX:        50,
			spriteIndex:   2,
			spriteX:       20, // higher X loses
			expectedClaim: false,
			expectedOwner: 3,
		},
		{
			name: "same X - lower OAM index wins",
			setup: func(b *SpritePriorityBuffer) {
				b.Clear()
				b.ownerIndex[50] = 5
				b.ownerX[50] = 20
			},
			pixelX:        50,
			spriteIndex:   3, // lower index wins
			spriteX:       20,
			expectedClaim: true,
			expectedOwner: 3,
		},
		{
			name: "same X - higher OAM index loses",
			setup: func(b *SpritePriorityBuffer) {
				b.Clear()
				b.ownerIndex[50] = 3
				b.ownerX[50] = 20
			},
			pixelX:        50,
			spriteIndex:   5, // higher index loses
			spriteX:       20,
			expectedClaim: false,
			expectedOwner: 3,
		},
		{
			name: "out of bounds - negative X",
			setup: func(b *SpritePriorityBuffer) {
				b.Clear()
			},
			pixelX:        -1,
			spriteIndex:   2,
			spriteX:       20,
			expectedClaim: false,
			expectedOwner: -1,
		},
		{
			name: "out of bounds - X >= width",
			setup: func(b *SpritePriorityBuffer) {
				b.Clear()
			},
			pixelX:        FramebufferWidth,
			spriteIndex:   2,
			spriteX:       20,
			expectedClaim: false,
			expectedOwner: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := &SpritePriorityBuffer{}
			tt.setup(buffer)

			claimed := buffer.TryClaimPixel(tt.pixelX, tt.spriteIndex, tt.spriteX)
			assert.Equal(t, tt.expectedClaim, claimed, "claim result mismatch")

			owner := buffer.GetOwner(tt.pixelX)
			assert.Equal(t, tt.expectedOwner, owner, "owner mismatch")
		})
	}
}

func TestSpritePriorityBuffer_GetOwner(t *testing.T) {
	buffer := &SpritePriorityBuffer{}
	buffer.Clear()

	// set some owners
	buffer.ownerIndex[0] = 5
	buffer.ownerIndex[50] = 3
	buffer.ownerIndex[159] = 7

	// test getting owners
	assert.Equal(t, 5, buffer.GetOwner(0))
	assert.Equal(t, 3, buffer.GetOwner(50))
	assert.Equal(t, 7, buffer.GetOwner(159))
	assert.Equal(t, -1, buffer.GetOwner(100)) // unowned

	// test out of bounds
	assert.Equal(t, -1, buffer.GetOwner(-1))
	assert.Equal(t, -1, buffer.GetOwner(FramebufferWidth))
}

func TestSpritePriorityBuffer_CompleteScenario(t *testing.T) {
	// simulate a realistic sprite overlap scenario
	buffer := &SpritePriorityBuffer{}
	buffer.Clear()

	// sprite 0 at X=20 covers pixels 20-27
	for i := 0; i < 8; i++ {
		buffer.TryClaimPixel(20+i, 0, 20)
	}

	// sprite 1 at X=15 covers pixels 15-22 (overlaps with sprite 0)
	// should win pixels 15-22 due to lower X
	for i := 0; i < 8; i++ {
		buffer.TryClaimPixel(15+i, 1, 15)
	}

	// sprite 2 at X=15 covers pixels 15-22 (same X as sprite 1)
	// should lose to sprite 1 due to higher OAM index
	for i := 0; i < 8; i++ {
		buffer.TryClaimPixel(15+i, 2, 15)
	}

	// verify ownership
	// pixels 15-19: sprite 1 (lowest X)
	for i := 15; i < 20; i++ {
		assert.Equal(t, 1, buffer.GetOwner(i), "pixel %d should be owned by sprite 1", i)
	}

	// pixels 20-22: sprite 1 (lower X than sprite 0)
	for i := 20; i <= 22; i++ {
		assert.Equal(t, 1, buffer.GetOwner(i), "pixel %d should be owned by sprite 1", i)
	}

	// pixels 23-27: sprite 0 (no overlap)
	for i := 23; i <= 27; i++ {
		assert.Equal(t, 0, buffer.GetOwner(i), "pixel %d should be owned by sprite 0", i)
	}
}

// Test documentation Example 1: Different X coordinates
func TestSpritePriorityBuffer_DocExample1(t *testing.T) {
	// Example 1 from documentation:
	// Sprite 0: X=5, OAM=0, covers pixels 5-12
	// Sprite 1: X=10, OAM=1, covers pixels 10-17
	// Expected: Sprite 0 wins all its pixels due to lower X
	
	buffer := &SpritePriorityBuffer{}
	buffer.Clear()
	
	// sprite 0 at X=5
	for i := 0; i < 8; i++ {
		buffer.TryClaimPixel(5+i, 0, 5)
	}
	
	// sprite 1 at X=10 
	for i := 0; i < 8; i++ {
		buffer.TryClaimPixel(10+i, 1, 10)
	}
	
	// verify: pixels 5-9 owned by sprite 0 (no overlap)
	for i := 5; i <= 9; i++ {
		assert.Equal(t, 0, buffer.GetOwner(i), "pixel %d should be owned by sprite 0", i)
	}
	
	// verify: pixels 10-12 still owned by sprite 0 (wins overlap due to lower X)
	for i := 10; i <= 12; i++ {
		assert.Equal(t, 0, buffer.GetOwner(i), "pixel %d should be owned by sprite 0 (lower X)", i)
	}
	
	// verify: pixels 13-17 owned by sprite 1 (no overlap)
	for i := 13; i <= 17; i++ {
		assert.Equal(t, 1, buffer.GetOwner(i), "pixel %d should be owned by sprite 1", i)
	}
}

// Test documentation Example 2: Same X coordinates with priority
func TestSpritePriorityBuffer_DocExample2(t *testing.T) {
	// Example 2 from documentation:
	// Sprite 1: X=12, OAM=1, covers pixels 12-19 
	// Sprite 3: X=12, OAM=3, covers pixels 12-19
	// Sprite 5: X=10, OAM=5, covers pixels 10-17
	// Expected: 
	// - Pixels 10-17: Sprite 5 (lowest X)
	// - Pixels 18-19: Sprite 1 (same X as 3, lower OAM)
	
	buffer := &SpritePriorityBuffer{}
	buffer.Clear()
	
	// add sprites in OAM order (1, 3, 5)
	// sprite 1 at X=12
	for i := 0; i < 8; i++ {
		buffer.TryClaimPixel(12+i, 1, 12)
	}
	
	// sprite 3 at X=12
	for i := 0; i < 8; i++ {
		buffer.TryClaimPixel(12+i, 3, 12)
	}
	
	// sprite 5 at X=10
	for i := 0; i < 8; i++ {
		buffer.TryClaimPixel(10+i, 5, 10)
	}
	
	// verify: pixels 10-11 owned by sprite 5 (no overlap)
	for i := 10; i <= 11; i++ {
		assert.Equal(t, 5, buffer.GetOwner(i), "pixel %d should be owned by sprite 5", i)
	}
	
	// verify: pixels 12-17 owned by sprite 5 (lowest X wins)
	for i := 12; i <= 17; i++ {
		assert.Equal(t, 5, buffer.GetOwner(i), "pixel %d should be owned by sprite 5 (lowest X)", i)
	}
	
	// verify: pixels 18-19 owned by sprite 1 (lower OAM than sprite 3)
	for i := 18; i <= 19; i++ {
		assert.Equal(t, 1, buffer.GetOwner(i), "pixel %d should be owned by sprite 1 (lower OAM)", i)
	}
}
