package main

import "math/rand/v2"

var forkNameAdjectives = []string{
	"swift", "calm", "bold", "bright", "keen",
	"fair", "wise", "warm", "cool", "neat",
	"fine", "true", "pure", "vast", "rare",
	"soft", "firm", "deep", "wild", "free",
}

var forkNameColors = []string{
	"red", "blue", "green", "amber", "coral",
	"ivory", "jade", "lime", "mint", "navy",
	"plum", "rose", "ruby", "sage", "teal",
}

const forkNameSuffixChars = "0123456789aeiou"

func generateRandomForkName() string {
	adjective := forkNameAdjectives[rand.IntN(len(forkNameAdjectives))]
	color := forkNameColors[rand.IntN(len(forkNameColors))]
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = forkNameSuffixChars[rand.IntN(len(forkNameSuffixChars))]
	}
	return adjective + "-" + color + "-" + string(suffix)
}
