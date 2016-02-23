package skein

import (
	"github.com/jurgen-kluft/Case/hashing/threefish"
)

type skeinConfiguration struct {
	numStateWords int
	configValue   []uint64
	// Set the state size for the configuration
	configString []uint64
}

func newSkeinConfiguration(sk *Skein) *skeinConfiguration {
	s := new(skeinConfiguration)
	s.numStateWords = sk.getNumberCipherStateWords()
	s.configValue = make([]uint64, s.numStateWords)
	s.configString = make([]uint64, s.numStateWords)
	s.configString[1] = uint64(sk.getHashSize())
	return s
}

func (c *skeinConfiguration) generateConfiguration() {

	tweak := newUbiTweak()

	// Initialize the tweak value
	tweak.startNewBlockType(uint64(Config))
	tweak.setFinalBlock(true)
	tweak.setBitsProcessed(32)

	cipher, _ := threefish.NewSize(c.numStateWords * 64)
	cipher.SetTweak(tweak.getTweak())
	cipher.Encrypt64(c.configValue, c.configString)

	c.configValue[0] ^= c.configString[0]
	c.configValue[1] ^= c.configString[1]
	c.configValue[2] ^= c.configString[2]
}

func (c *skeinConfiguration) generateConfigurationState(initialState []uint64) {

	tweak := newUbiTweak()

	// Initialize the tweak value
	tweak.startNewBlockType(uint64(Config))
	tweak.setFinalBlock(true)
	tweak.setBitsProcessed(32)

	cipher, _ := threefish.New64(initialState, tweak.getTweak())
	cipher.Encrypt64(c.configValue, c.configString)

	c.configValue[0] ^= c.configString[0]
	c.configValue[1] ^= c.configString[1]
	c.configValue[2] ^= c.configString[2]
}

func (c *skeinConfiguration) setSchema(schema []byte) {

	n := c.configString[0]

	// Clear the schema bytes
	n &^= 0xffffffff
	// Set schema bytes
	n = uint64(schema[3]) << 24
	n |= uint64(schema[2]) << 16
	n |= uint64(schema[1]) << 8
	n |= uint64(schema[0])

	c.configString[0] = n
}

func (c *skeinConfiguration) setVersion(version int) {
	c.configString[0] &^= uint64(0x03) << 32
	c.configString[0] |= uint64(version) << 32
}

func (c *skeinConfiguration) setTreeLeafSize(size byte) {
	c.configString[2] &^= uint64(0xff)
	c.configString[2] |= uint64(size)
}

func (c *skeinConfiguration) setTreeFanOutSize(size byte) {
	c.configString[2] &^= uint64(0xff) << 8
	c.configString[2] |= uint64(size) << 8
}

func (c *skeinConfiguration) setMaxTreeHeight(height byte) {
	c.configString[2] &^= uint64(0xff) << 16
	c.configString[2] |= uint64(height) << 16
}
