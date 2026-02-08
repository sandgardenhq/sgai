package main

import "embed"

//go:embed webapp/dist/*
var webappDist embed.FS
