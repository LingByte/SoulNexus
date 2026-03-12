package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmojiConstants(t *testing.T) {
	// Test face emojis
	assert.Equal(t, "\\U0001F603", Happy, "Happy should be '\\U0001F603'")
	assert.Equal(t, "\\U0001F604", Smile, "Smile should be '\\U0001F604'")
	assert.Equal(t, "\\U0001F60D", LoveEye, "LoveEye should be '\\U0001F60D'")
	assert.Equal(t, "\\U0001F60D", StarkStarkEye, "StarkStarkEye should be '\\U0001F60D'")
	assert.Equal(t, "\uF622", Cry, "Cry should be '\uF622'")
	assert.Equal(t, "\\U0001F610", Neutral, "Neutral should be '\\U0001F610'")
	assert.Equal(t, "\\U0001F615", Confused, "Confused should be '\\U0001F615'")
	assert.Equal(t, "\\U0001F61F", Worried, "Worried should be '\\U0001F61F'")
	assert.Equal(t, "\\U0001F620", Angry, "Angry should be '\\U0001F620'")
	assert.Equal(t, "\\U0001F621", Rage, "Rage should be '\\U0001F621'")
	assert.Equal(t, "\\U0001F613", Sweat, "Sweat should be '\\U0001F613'")
	assert.Equal(t, "\\U0001F62B", Tired, "Tired should be '\\U0001F62B'")
	assert.Equal(t, "\\U0001F62A", Sleepy, "Sleepy should be '\\U0001F62A'")
	assert.Equal(t, "\\U0001F612", Unamused, "Unamused should be '\\U0001F612'")
	assert.Equal(t, "\\U0001F644", RollingEyes, "RollingEyes should be '\\U0001F644'")
	assert.Equal(t, "\\U0001F60A", Blush, "Blush should be '\\U0001F60A'")
	assert.Equal(t, "\\U0001F601", Grin, "Grin should be '\\U0001F601'")
	assert.Equal(t, "\\U0001F602", Joy, "Joy should be '\\U0001F602'")
	assert.Equal(t, "\\U0001F642", SlightSmile, "SlightSmile should be '\\U0001F642'")
	assert.Equal(t, "\\U0001F609", Wink, "Wink should be '\\U0001F609'")
	assert.Equal(t, "\\U0001F618", Kiss, "Kiss should be '\\U0001F618'")
	assert.Equal(t, "\\U0001F60D", HeartEyes, "HeartEyes should be '\\U0001F60D'")
	assert.Equal(t, "\\U0001F929", StarStruck, "StarStruck should be '\\U0001F929'")
	assert.Equal(t, "\\U0001F914", Thinking, "Thinking should be '\\U0001F914'")
	assert.Equal(t, "\\U0001F970", Yawning, "Yawning should be '\\U0001F970'")

	// Test hand emojis
	assert.Equal(t, "\\U0001F44D", ThumbsUp, "ThumbsUp should be '\\U0001F44D'")
	assert.Equal(t, "\\U0001F44E", ThumbsDown, "ThumbsDown should be '\\U0001F44E'")
	assert.Equal(t, "\\U0000270C", Victory, "Victory should be '\\U0000270C'")
	assert.Equal(t, "\\U0001F919", CallMe, "CallMe should be '\\U0001F919'")
	assert.Equal(t, "\\U0001F44B", Wave, "Wave should be '\\U0001F44B'")
	assert.Equal(t, "\\U0001F64F", FoldedHands, "FoldedHands should be '\\U0001F64F'")
	assert.Equal(t, "\\U0001F44F", Clap, "Clap should be '\\U0001F44F'")
	assert.Equal(t, "\\U0000270B", RaisedHand, "RaisedHand should be '\\U0000270B'")
	assert.Equal(t, "\\U0001F44C", OkHand, "OkHand should be '\\U0001F44C'")

	// Test nature emojis
	assert.Equal(t, "\\U00002600", Sun, "Sun should be '\\U00002600'")
	assert.Equal(t, "\\U0001F319", Moon, "Moon should be '\\U0001F319'")
	assert.Equal(t, "\\U00002601", Cloud, "Cloud should be '\\U00002601'")
	assert.Equal(t, "\\U0001F327", Rain, "Rain should be '\\U0001F327'")
	assert.Equal(t, "\\U000026A1", Lightning, "Lightning should be '\\U000026A1'")
	assert.Equal(t, "\\U00002744", Snowflake, "Snowflake should be '\\U00002744'")
	assert.Equal(t, "\\U0001F525", Fire, "Fire should be '\\U0001F525'")
	assert.Equal(t, "\\U0001F332", Tree, "Tree should be '\\U0001F332'")
	assert.Equal(t, "\\U0001F490", Flower, "Flower should be '\\U0001F490'")

	// Test animal emojis
	assert.Equal(t, "\\U0001F415", Dog, "Dog should be '\\U0001F415'")
	assert.Equal(t, "\\U0001F408", Cat, "Cat should be '\\U0001F408'")
	assert.Equal(t, "\\U0001F412", Monkey, "Monkey should be '\\U0001F412'")
	assert.Equal(t, "\\U0001F981", Lion, "Lion should be '\\U0001F981'")
	assert.Equal(t, "\\U0001F405", Tiger, "Tiger should be '\\U0001F405'")
	assert.Equal(t, "\\U0001F418", Elephant, "Elephant should be '\\U0001F418'")
	assert.Equal(t, "\\U0001F43C", Panda, "Panda should be '\\U0001F43C'")
	assert.Equal(t, "\\U0001F40E", Horse, "Horse should be '\\U0001F40E'")
	assert.Equal(t, "\\U0001F404", Cow, "Cow should be '\\U0001F404'")
	assert.Equal(t, "\\U0001F41F", Fish, "Fish should be '\\U0001F41F'")

	// Test food emojis
	assert.Equal(t, "\\U0001F355", Pizza, "Pizza should be '\\U0001F355'")
	assert.Equal(t, "\\U0001F354", Burger, "Burger should be '\\U0001F354'")
	assert.Equal(t, "\\U0001F363", Sushi, "Sushi should be '\\U0001F363'")
	assert.Equal(t, "\\U0001F32E", Taco, "Taco should be '\\U0001F32E'")
	assert.Equal(t, "\\U0001F32D", Hotdog, "Hotdog should be '\\U0001F32D'")
	assert.Equal(t, "\\U0001F370", Cake, "Cake should be '\\U0001F370'")
	assert.Equal(t, "\\U0001F366", IceCream, "IceCream should be '\\U0001F366'")
	assert.Equal(t, "\\U00002615", Coffee, "Coffee should be '\\U00002615'")
	assert.Equal(t, "\\U0001F37A", Beer, "Beer should be '\\U0001F37A'")
	assert.Equal(t, "\\U0001F377", WineGlass, "WineGlass should be '\\U0001F377'")

	// Test object emojis
	assert.Equal(t, "\\U0001F4BB", Computer, "Computer should be '\\U0001F4BB'")
	assert.Equal(t, "\\U0001F4F1", Mobile, "Mobile should be '\\U0001F4F1'")
	assert.Equal(t, "\\U0001F4F7", Camera, "Camera should be '\\U0001F4F7'")
	assert.Equal(t, "\\U0001F3A7", Headphones, "Headphones should be '\\U0001F3A7'")
	assert.Equal(t, "\\U0001F6E0", Tools, "Tools should be '\\U0001F6E0'")
	assert.Equal(t, "\\U0001F528", Hammer, "Hammer should be '\\U0001F528'")
	assert.Equal(t, "\\U0001F527", Wrench, "Wrench should be '\\U0001F527'")
	assert.Equal(t, "\\U00002699", Gear, "Gear should be '\\U00002699'")
	assert.Equal(t, "\\U0001F52C", Microscope, "Microscope should be '\\U0001F52C'")
	assert.Equal(t, "\\U0001F680", Rocket, "Rocket should be '\\U0001F680'")

	// Test symbol emojis
	assert.Equal(t, "\\U00002705", Checkmark, "Checkmark should be '\\U00002705'")
	assert.Equal(t, "\\U0000274C", CrossMark, "CrossMark should be '\\U0000274C'")
	assert.Equal(t, "\\U00002139", Info, "Info should be '\\U00002139'")
	assert.Equal(t, "\\U000026A0", Warning, "Warning should be '\\U000026A0'")
	assert.Equal(t, "\\U0001F6AB", NoEntry, "NoEntry should be '\\U0001F6AB'")
	assert.Equal(t, "\\U0001F514", Bell, "Bell should be '\\U0001F514'")
	assert.Equal(t, "\\U0001F512", Lock, "Lock should be '\\U0001F512'")
	assert.Equal(t, "\\U0001F513", Unlock, "Unlock should be '\\U0001F513'")
}
