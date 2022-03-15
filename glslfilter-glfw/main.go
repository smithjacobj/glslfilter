package main

import (
	"image"
	"image/png"
	_ "image/png"
	"os"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/smithjacobj/glslfilter"
	"github.com/smithjacobj/glslfilter/util"
)

const AppName = "GLSL Filter"

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

var showResult bool = true

func init() {
	fileInfo, err := os.Stdout.Stat()
	util.Invariant(err)
	// if we're just in a terminal and not piped, show the window
	showResult = fileInfo.Mode()&os.ModeCharDevice != 0
}

func main() {
	util.Invariant(glfw.Init())
	defer glfw.Terminate()

	if showResult {
		glfw.WindowHint(glfw.Visible, glfw.True)
	} else {
		glfw.WindowHint(glfw.Visible, glfw.False)
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	definition, err := glslfilter.LoadDefinitionFromStdin()
	util.Invariant(err)

	window, err := glfw.CreateWindow(definition.Render.Width, definition.Render.Height, AppName, nil, nil)
	util.Invariant(err)
	window.MakeContextCurrent()

	engine, err := glslfilter.NewEngine(image.Rect(0, 0, definition.Render.Width, definition.Render.Height), true, showResult)
	util.Invariant(err)

	stages := []*glslfilter.FilterStage{}
	for _, stageDefinition := range definition.Stages {
		fragmentShaderSource, err := glslfilter.LoadFragmentShader(stageDefinition.FragmentShaderPath)
		util.Invariant(err)

		textures := []glslfilter.Texture{}
		for _, textureDefinition := range stageDefinition.Textures {
			textureRGBA, err := glslfilter.LoadTextureData(textureDefinition.Path)
			util.Invariant(err)
			textures = append(textures, glslfilter.Texture{Data: textureRGBA, Filter: int32(textureDefinition.Filter)})
		}

		stage, err := glslfilter.NewFilterStage(fragmentShaderSource, textures)
		util.Invariant(err)

		stages = append(stages, stage)
	}

	err = engine.Init(stages)
	util.Invariant(err)

	engine.Render()
	window.SwapBuffers()

	if !showResult {
		imageData := engine.GetLastRenderImage()
		err := png.Encode(os.Stdout, imageData)
		util.Invariant(err)
	}

	for showResult && !window.ShouldClose() {
		glfw.PollEvents()
	}
}
